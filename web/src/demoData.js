const t = (v) => ({ k: 'text', v })
const kw = (v) => ({ k: 'kw', v })
const id = (v, ref) => ({ k: 'tok', v, ref })

export const diff = [
  {
    hunk: '@@ -18,7 +18,7 @@ func (s *Service) Charge(ctx context.Context, inv *Invoice) error {',
  },
  { ln: 18, kind: 'ctx', seg: [kw('if'), t(' inv == nil {')] },
  { ln: 19, kind: 'ctx', seg: [t('\t'), kw('return'), t(' ErrNoInvoice')] },
  { ln: 20, kind: 'ctx', seg: [t('}')] },
  {
    ln: 21,
    kind: 'del',
    seg: [
      t('total := inv.Subtotal + inv.Subtotal*'),
      id('taxRate', 'taxRate'),
    ],
  },
  {
    ln: 21,
    kind: 'add',
    seg: [
      t('total := '),
      id('getSubtotal', 'getSubtotal'),
      t('(inv) + '),
      id('applyTax', 'applyTax'),
      t('(inv.Subtotal)'),
    ],
  },
  { ln: 22, kind: 'ctx', seg: [kw('if'), t(' total <= 0 {')] },
  { ln: 23, kind: 'ctx', seg: [t('\t'), kw('return'), t(' ErrEmptyInvoice')] },
  { hunk: '@@ -52,0 +53,8 @@' },
  {
    ln: 53,
    kind: 'add',
    seg: [
      kw('func '),
      id('getSubtotal', 'getSubtotal'),
      t('(inv *'),
      id('Invoice', 'Invoice'),
      t(') int64 {'),
    ],
  },
  { ln: 54, kind: 'add', seg: [t('\t'), kw('var'), t(' sum int64')] },
  {
    ln: 55,
    kind: 'add',
    seg: [
      t('\t'),
      kw('for'),
      t(' _, li := '),
      kw('range'),
      t(' inv.'),
      id('LineItems', 'LineItems'),
      t(' {'),
    ],
  },
  {
    ln: 56,
    kind: 'add',
    seg: [t('\t\tsum += li.'), id('AmountCents', 'AmountCents')],
  },
  { ln: 57, kind: 'add', seg: [t('\t}')] },
  {
    ln: 58,
    kind: 'add',
    seg: [
      t('\t'),
      kw('return'),
      t(' '),
      id('roundCents', 'roundCents'),
      t('(sum)'),
    ],
  },
  { ln: 59, kind: 'add', seg: [t('}')] },
]

export const tokens = {
  getSubtotal: {
    name: 'getSubtotal',
    count: 1,
    files: 1,
    flag: {
      kind: 'convention',
      title: 'naming convention',
      body: 'This repo names this operation calculateSubtotal in 12 places. getSubtotal appears once — here.',
    },
    usages: [
      {
        path: 'internal/billing/invoice.go',
        line: 53,
        code: 'func getSubtotal(inv *Invoice) int64 {',
        self: true,
      },
    ],
    neighbours: [
      {
        path: 'internal/billing/totals.go',
        line: 22,
        code: 'func calculateSubtotal(items []LineItem) int64 {',
      },
      {
        path: 'internal/billing/order.go',
        line: 88,
        code: 'sub := calculateSubtotal(o.LineItems)',
      },
      {
        path: 'internal/reporting/monthly.go',
        line: 141,
        code: 'total += calculateSubtotal(inv.LineItems)',
      },
    ],
  },
  roundCents: {
    name: 'roundCents',
    count: 0,
    files: 0,
    flag: {
      kind: 'novel',
      title: 'novel identifier',
      body: 'roundCents appears nowhere else in this repository. Either it is genuinely new, or it is a typo of something that exists.',
    },
    usages: [],
    neighbours: [
      {
        path: 'internal/money/round.go',
        line: 14,
        code: 'func RoundCents(v int64) int64 {',
      },
    ],
  },
  Invoice: {
    name: 'Invoice',
    count: 34,
    files: 9,
    usages: [
      {
        path: 'internal/billing/invoice.go',
        line: 12,
        code: 'type Invoice struct {',
      },
      {
        path: 'internal/billing/service.go',
        line: 47,
        code: 'func (s *Service) Charge(ctx context.Context, inv *Invoice) error {',
      },
      {
        path: 'internal/billing/order.go',
        line: 31,
        code: 'inv := &Invoice{CustomerID: o.CustomerID}',
      },
      {
        path: 'internal/api/invoices.go',
        line: 76,
        code: 'var body Invoice',
      },
      {
        path: 'internal/reporting/monthly.go',
        line: 58,
        code: 'for _, inv := range invoices {',
      },
      {
        path: 'internal/store/postgres.go',
        line: 204,
        code: 'func (p *Postgres) SaveInvoice(inv *Invoice) error {',
      },
    ],
  },
  applyTax: {
    name: 'applyTax',
    count: 6,
    files: 3,
    usages: [
      {
        path: 'internal/billing/totals.go',
        line: 40,
        code: 'func applyTax(cents int64) int64 {',
      },
      {
        path: 'internal/billing/order.go',
        line: 92,
        code: 'withTax := applyTax(sub)',
      },
      {
        path: 'internal/billing/totals_test.go',
        line: 18,
        code: 'got := applyTax(1000)',
      },
    ],
  },
  taxRate: {
    name: 'taxRate',
    count: 8,
    files: 2,
    usages: [
      {
        path: 'internal/billing/totals.go',
        line: 9,
        code: 'const taxRate = 0.0825',
      },
      {
        path: 'internal/billing/totals.go',
        line: 41,
        code: 'return int64(float64(cents) * taxRate)',
      },
      {
        path: 'internal/billing/totals_test.go',
        line: 31,
        code: 'want := int64(float64(1000) * taxRate)',
      },
    ],
  },
  LineItems: {
    name: 'LineItems',
    count: 17,
    files: 6,
    usages: [
      {
        path: 'internal/billing/invoice.go',
        line: 16,
        code: 'LineItems []LineItem',
      },
      {
        path: 'internal/billing/order.go',
        line: 88,
        code: 'sub := calculateSubtotal(o.LineItems)',
      },
      {
        path: 'internal/api/invoices.go',
        line: 91,
        code: 'if len(body.LineItems) == 0 {',
      },
    ],
  },
  AmountCents: {
    name: 'AmountCents',
    count: 11,
    files: 4,
    usages: [
      {
        path: 'internal/billing/invoice.go',
        line: 24,
        code: 'AmountCents int64',
      },
      {
        path: 'internal/billing/totals.go',
        line: 25,
        code: 'sum += it.AmountCents',
      },
      {
        path: 'internal/store/postgres.go',
        line: 231,
        code: 'row.Scan(&li.AmountCents)',
      },
    ],
  },
}

export const signals = [
  {
    tag: 'convention',
    accent: 'var(--tint-3)',
    title: 'Naming that breaks the pattern',
    body: 'A new identifier that is one small step from a name the codebase already uses consistently. Scoped to the same directory, because conventions are local.',
    example: 'adds  getSubtotal\nrepo   calculateSubtotal ×12',
  },
  {
    tag: 'blast radius',
    accent: 'var(--tint-1)',
    title: 'How far the change reaches',
    body: 'The identifiers this diff touches, ranked by how much of the codebase depends on them. Not navigation — a reason to read more carefully.',
    example: 'Invoice        34 usages · 9 files\nAmountCents    11 usages · 4 files',
  },
  {
    tag: 'novel',
    accent: 'var(--tint-4)',
    title: 'Names that appear nowhere else',
    body: 'Zero occurrences outside the diff. Sometimes genuinely new, sometimes a typo of something that already exists one directory over.',
    example: 'roundCents     0 usages\nnear  money.RoundCents',
  },
  {
    tag: 'duplication',
    accent: 'var(--tint-5)',
    title: 'Code you already have',
    body: 'Added blocks that closely match existing code, matched on normalised tokens so renamed copies still register.',
    example: 'lines 53–59  ≈  totals.go:22–28\n            91% token overlap',
  },
]

export const perf = [
  { num: '1.3', unit: ' GB/s', label: 'Corpus line scan', sub: 'MapBoundaries' },
  { num: '43', unit: ' ns', label: 'Query miss', sub: '1 allocation' },
  { num: '20', unit: ' ns', label: 'Line retrieval', sub: 'mmap, zero copy' },
  { num: '0', unit: '', label: 'Inference calls', sub: 'deterministic' },
]
