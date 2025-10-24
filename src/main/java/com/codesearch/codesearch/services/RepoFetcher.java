package com.codesearch.codesearch.services;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.eclipse.jgit.api.Git;
import org.springframework.stereotype.Service;

import java.io.File;
import java.util.ArrayList;
import java.util.List;

@Service
public class RepoFetcher {

    private final GithubService githubService;
    private final ObjectMapper objectMapper = new ObjectMapper();
    private final List<String> repos = new ArrayList<>();

    public RepoFetcher(GithubService githubService) {
        this.githubService = githubService;
    }


    public List<String> fetchRepos(String token) {
        repos.clear();
        try {
            String response = githubService.getUserRepos(token);
            if (response == null || response.isEmpty()) {
                return repos;
            }

            JsonNode root = objectMapper.readTree(response);
            if (root.isArray()) {
                for (JsonNode repoNode : root) {
                    JsonNode nameNode = repoNode.get("name");
                    JsonNode cloneUrlNode = repoNode.get("clone_url");

                    if (nameNode != null && cloneUrlNode != null) {
                        String name = nameNode.asText();
                        String cloneUrl = cloneUrlNode.asText();
                        repos.add(name);
                        cloneRepository(name, cloneUrl, token);
                    }
                }
            }
        } catch (Exception e) {
         //
        }
        return new ArrayList<>(repos);
    }

    private void cloneRepository(String name, String cloneUrl, String token) {
        try {
            // Insert token into HTTPS URL for authentication
            String authenticatedUrl = cloneUrl.replace("https://", "https://" + token + "@");

            // Local target directory
            String clonePath = "/Users/zesanrahim/dev/codesearch-repo-fetch";
            File targetDir = new File(clonePath, name);

            if (targetDir.exists()) {

                return;
            }


            // Use try-with-resources to auto-close Git instance
            try (Git git = Git.cloneRepository()
                    .setURI(authenticatedUrl)
                    .setDirectory(targetDir)
                    .setDepth(1)
                    .call()) {

            }

        } catch (Exception e) {
            // add error
        }
    }

    public List<String> getRepos() {
        return new ArrayList<>(repos);
    }
}
