package com.codesearch.codesearch.services;

import com.codesearch.codesearch.models.Github;
import com.codesearch.codesearch.repositories.GithubRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.Optional;

@Service
public class ElasticService {
    @Autowired
    private GithubRepository githubRepository;

    public void testES() {
        // Simple smoke test for Elasticsearch @Document (Github)
        Github doc = new Github();
        doc.setId("test-1");
        doc.setRepoName("codesearch");
        doc.setFilePath("/examples/ElasticService.java");
        doc.setFilename("ElasticService.java");
        doc.setLanguage("java");
        doc.setContent("public class ElasticService {}");
        doc.setContentHash("hash-test-1");

        githubRepository.save(doc);

        Optional<Github> byHash = githubRepository.findByContentHash("hash-test-1");
        System.out.println(byHash.orElse(null));
    }
}

