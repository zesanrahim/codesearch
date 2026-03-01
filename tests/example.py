import os
import json
from pathlib import Path
import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
import joblib

from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import StratifiedKFold
from sklearn.metrics import (
    f1_score,
    precision_score,
    recall_score,
    confusion_matrix,
    roc_auc_score,
    average_precision_score,
    log_loss,
    roc_curve,
    precision_recall_curve,
    auc,
    classification_report,
)
from sklearn.preprocessing import label_binarize

try:
    import tests.ml_training as ml_training
except Exception:
    import importlib.util
    spec = importlib.util.spec_from_file_location("ml_training", "tests/ml_training.py")
    ml_training = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(ml_training)


def _ensure_dir(d):
    os.makedirs(d, exist_ok=True)


def plot_confusion(cm, labels, out_path, normalize=False):
    if normalize:
        cm = cm.astype(float) / (cm.sum(axis=1, keepdims=True) + 1e-12)

    fig, ax = plt.subplots(figsize=(6, 5))
    im = ax.imshow(cm, interpolation="nearest", cmap=plt.cm.Blues)
    ax.figure.colorbar(im, ax=ax)
    ax.set(xticks=np.arange(len(labels)), yticks=np.arange(len(labels)), xticklabels=labels, yticklabels=labels, ylabel="True label", xlabel="Predicted label")
    plt.setp(ax.get_xticklabels(), rotation=45, ha="right", rotation_mode="anchor")

    fmt = ".2f" if normalize else "d"
    thresh = cm.max() / 2.0
    for i in range(cm.shape[0]):
        for j in range(cm.shape[1]):
            val = cm[i, j]
            ax.text(j, i, format(val, fmt), ha="center", va="center", color="white" if val > thresh else "black")

    fig.tight_layout()
    fig.savefig(out_path)
    plt.close(fig)


def plot_feature_importance(model, feature_names, out_path, top_n=20):
    if not hasattr(model, "feature_importances_"):
        return

    fi = pd.DataFrame({
        "feature": feature_names,
        "importance": model.feature_importances_
    }).sort_values("importance", ascending=True).tail(top_n)

    fig, ax = plt.subplots(figsize=(10, 8))
    ax.barh(fi["feature"], fi["importance"], color="teal")
    ax.set_title(f"Top {top_n} Feature Importances")
    ax.set_xlabel("Importance")
    ax.set_ylabel("Feature")
    
    fig.tight_layout()
    fig.savefig(out_path)
    plt.close(fig)





def run_cv(X, y, k=5, save_dir="csvs/artifacts", random_state=42, save_models=True):
    save_dir = Path(save_dir)
    _ensure_dir(save_dir)
    _ensure_dir(save_dir / "plots")
    _ensure_dir(save_dir / "metrics")
    _ensure_dir(save_dir / "confusion_matrices")
    _ensure_dir(save_dir / "models")

    labels = sorted(y.unique())
    n_classes = len(labels)

    skf = StratifiedKFold(n_splits=k, shuffle=True, random_state=random_state)

    fold_metrics = []
    cms = []
    all_y_true = []
    all_y_pred = []

    for fold, (tr_idx, te_idx) in enumerate(skf.split(X, y), start=1):
        Xtr, Xte = X.iloc[tr_idx], X.iloc[te_idx]
        ytr, yte = y.iloc[tr_idx], y.iloc[te_idx]

        model = RandomForestClassifier(n_estimators=200, class_weight="balanced", random_state=random_state)
        model.fit(Xtr, ytr)

        ypred = model.predict(Xte)
        
        all_y_true.extend(yte)
        all_y_pred.extend(ypred)

        yprob = None
        if hasattr(model, "predict_proba"):
            yprob = model.predict_proba(Xte)

        m = {
            "f1_weighted": float(f1_score(yte, ypred, average="weighted")),
            "precision_weighted": float(precision_score(yte, ypred, average="weighted", zero_division=0)),
            "recall_weighted": float(recall_score(yte, ypred, average="weighted", zero_division=0)),
        }

        if yprob is not None:
            try:
                m["log_loss"] = float(log_loss(yte, yprob))
            except Exception:
                m["log_loss"] = None

        if yprob is not None and n_classes > 1:
            try:
                y_true_bin = label_binarize(yte, classes=labels)
                m["roc_auc_ovr"] = float(roc_auc_score(y_true_bin, yprob, average="macro", multi_class="ovr"))
            except Exception:
                m["roc_auc_ovr"] = None

            try:
                ap = []
                for c in range(yprob.shape[1]):
                    ap.append(average_precision_score(y_true_bin[:, c], yprob[:, c]))
                m["pr_auc_macro"] = float(np.mean(ap))
            except Exception:
                m["pr_auc_macro"] = None

        cm = confusion_matrix(yte, ypred, labels=labels)
        cms.append(cm)

        cm_csv = save_dir / "confusion_matrices" / f"confusion_fold_{fold}.csv"
        pd.DataFrame(cm, index=labels, columns=labels).to_csv(cm_csv)
        plot_confusion(cm, labels, save_dir / "confusion_matrices" / f"confusion_fold_{fold}.png", normalize=False)
        plot_confusion(cm, labels, save_dir / "confusion_matrices" / f"confusion_fold_{fold}_norm.png", normalize=True)

        if save_models:
            model_path = save_dir / "models" / f"model_fold_{fold}.pkl"
            try:
                joblib.dump(model, model_path)
            except Exception:
                pass



        fold_metrics.append(m)

    agg = {}
    for key in fold_metrics[0].keys():
        vals = [fm.get(key) for fm in fold_metrics if fm.get(key) is not None]
        if len(vals) == 0:
            agg[key] = {"mean": None, "std": None}
        else:
            agg[key] = {"mean": float(np.mean(vals)), "std": float(np.std(vals))}

    total_cm = np.sum(cms, axis=0)
    total_cm = np.sum(cms, axis=0)
    pd.DataFrame(total_cm, index=labels, columns=labels).to_csv(save_dir / "confusion_matrices" / "confusion_total.csv")
    plot_confusion(total_cm, labels, save_dir / "confusion_matrices" / "confusion_total.png", normalize=False)
    plot_confusion(total_cm, labels, save_dir / "confusion_matrices" / "confusion_total_norm.png", normalize=True)

    report = classification_report(all_y_true, all_y_pred, target_names=labels, zero_division=0)
    with open(save_dir / "metrics" / "classification_report.txt", "w") as f:
        f.write(report)



    out = {"per_fold": fold_metrics, "aggregate": agg, "labels": labels}
    with open(save_dir / "metrics" / "metrics.json", "w") as f:
        json.dump(out, f, indent=2)

    return out


def main(data_path="csvs/final_training_6.csv", k=5):
    df = ml_training.load_dataset(data_path)
    X, y = ml_training.prepare_data(df)
    return run_cv(X, y, k=k)


if __name__ == "__main__":
    import argparse

    p = argparse.ArgumentParser()
    p.add_argument("--data", default="csvs/final_training_6.csv")
    p.add_argument("--k", type=int, default=5)
    args = p.parse_args()
    print("Running CV evaluation... this may take a minute")
    res = main(args.data, args.k)
    print("Saved metrics to csvs/artifacts/metrics.json")