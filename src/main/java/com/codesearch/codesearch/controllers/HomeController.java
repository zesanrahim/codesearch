package com.codesearch.codesearch.controllers;

import com.codesearch.codesearch.services.GithubService;
import com.codesearch.codesearch.services.OAuthConnection;
import com.codesearch.codesearch.services.RepoFetcher;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.web.csrf.CsrfToken;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.ui.Model;

@Controller
public class HomeController {

    @Autowired
    private OAuthConnection oauthConnection;

    @Autowired
    private GithubService githubService;

    @Autowired
    private RepoFetcher repoFetcher;

    private final ObjectMapper mapper = new ObjectMapper();

    /**
     * Renders the home page with the user's GitHub repositories
     */
    @GetMapping("/")
    @ResponseBody
    public String home(OAuth2AuthenticationToken authentication, CsrfToken csrfToken, Model model) {
        if (authentication == null) {
            return """
                <html>
                <body style='font-family:Arial;'>
                ❌ Not authenticated.<br>
                Please <a href="/oauth2/authorization/github">log in with GitHub</a>.
                </body></html>
                """;
        }

        // 1️⃣ Get user access token
        String token = oauthConnection.handleLogin(authentication);

        // 2️⃣ Fetch repositories
        String reposJson = githubService.getUserRepos(token);
        StringBuilder htmlList = new StringBuilder("<ul>");

        try {
            JsonNode repos = mapper.readTree(reposJson);
            for (JsonNode repo : repos) {
                String name = repo.get("name").asText();
                String htmlUrl = repo.get("html_url").asText();
                htmlList.append("<li><a href=\"")
                        .append(htmlUrl)
                        .append("\" target=\"_blank\">")
                        .append(name)
                        .append("</a></li>");
            }
            htmlList.append("</ul>");
        } catch (Exception e) {
            htmlList = new StringBuilder("<p>⚠️ Failed to parse repositories.</p>");
            e.printStackTrace();
        }

        // 3️⃣ Button for repo downloading
        String downloadButton = String.format("""
            <form action="/download-repos" method="post">
                <input type="hidden" name="_csrf" value="%s" />
                <input type="hidden" name="token" value="%s" />
                <button type="submit"
                    style="background-color:#2da44e;color:white;
                    border:none;padding:10px 16px;border-radius:6px;
                    cursor:pointer;font-size:14px;">
                    ⬇️ Download All Repositories
                </button>
            </form>
            """, csrfToken.getToken(), token);

        // 4️⃣ Build final page
        return String.format("""
            <html>
            <head>
                <title>My GitHub Repositories</title>
                <style>
                    body { font-family: Arial, sans-serif; margin: 2rem; color: #24292f; background: #fafbfc; }
                    ul { list-style: none; padding: 0; }
                    li { margin: 8px 0; }
                    a { color: #0969da; text-decoration: none; }
                    a:hover { text-decoration: underline; }
                    button:hover { background-color:#2c974b; }
                </style>
            </head>
            <body>
                <h2>✅ Logged in with GitHub!</h2>
                <p>Your repositories:</p>
                %s
                <br><br>
                %s
                <br><a href="/logout">Logout</a>
            </body>
            </html>
            """, htmlList, downloadButton);
    }

    /**
     * Handles repository download POST request
     */
    @PostMapping("/download-repos")
    @ResponseBody
    public String downloadRepos(String token) {
        try {
            repoFetcher.fetchRepos(token);
            return """
                <html><body style='font-family:Arial;'>
                <h3>✅ Repositories are being downloaded!</h3>
                <p>They’ll appear in your local clone directory soon.</p>
                <a href="/">⬅️ Back</a>
                </body></html>
                """;
        } catch (Exception e) {
            return String.format("""
                <html><body style='font-family:Arial;'>
                <h3>❌ Failed to start repo download.</h3>
                <p>Error: %s</p>
                <a href="/">⬅️ Back</a>
                </body></html>
                """, e.getMessage());
        }
    }
}
