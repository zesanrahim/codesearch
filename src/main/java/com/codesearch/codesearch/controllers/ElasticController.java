package com.codesearch.codesearch.controllers;

import com.codesearch.codesearch.services.ElasticService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/elastic")
public class ElasticController {

    @Autowired
    private ElasticService elasticService;

    @GetMapping("/test")
    public String test() {
        elasticService.testES();
        return "✅ Elasticsearch test executed — check your console!";
    }
}
