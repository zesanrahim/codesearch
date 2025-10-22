package com.codesearch.codesearch.services;

import com.codesearch.codesearch.models.User;
import com.codesearch.codesearch.repositories.UserRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.oauth2.client.authentication.OAuth2AuthenticationToken;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.oauth2.client.*;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;
import java.util.Objects;

@RestController
@EnableWebSecurity
public class OAuthConnection {

    @Autowired
    private OAuth2AuthorizedClientService authorizedClientService;

    @Autowired
    private UserRepository userRepository;


    @Bean
    public SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        http
                .authorizeHttpRequests(auth -> auth

                        .anyRequest().authenticated()
                )
                .oauth2Login(oauth2 -> oauth2
                        .defaultSuccessUrl("/", true)
                )
                .logout(logout -> logout
                        .logoutSuccessUrl("/").permitAll()
                );
        return http.build();
    }

    private String getAccessToken(OAuth2AuthenticationToken authentication) {
        OAuth2AuthorizedClient client = authorizedClientService.loadAuthorizedClient(
                authentication.getAuthorizedClientRegistrationId(),
                authentication.getName());

        if (client == null) {
            return null;
        }
        // TODO: hash each token in the db
        return client.getAccessToken().getTokenValue();
    }


    private void saveOrUpdateUser(OAuth2AuthenticationToken authentication, String token) {

        Map<String, Object> attrs = authentication.getPrincipal().getAttributes();

        String username = (String) attrs.get("login");
        String email = (String) attrs.get("email");
        String tokenVal = getAccessToken(authentication);


        User user = userRepository.findByUsername(username).orElse(null);

        if (user == null) {

            user = new User();
            user.setEmail(email);
            user.setToken(tokenVal);
            user.setUsername(username);

        } else {

            boolean updated = false;

            if (!Objects.equals(email, user.getEmail())) {
                user.setEmail(email);
                updated = true;
            }

            if (!Objects.equals(tokenVal, user.getToken())) {
                user.setToken(tokenVal);
                updated = true;
            }

            if (updated) {
                userRepository.save(user);

            }

        }

    }


    public String handleLogin(OAuth2AuthenticationToken authentication) {
        String token = getAccessToken(authentication);
        if (token == null) return "No token found.";
        saveOrUpdateUser(authentication, token);
        return token;
    }
}
