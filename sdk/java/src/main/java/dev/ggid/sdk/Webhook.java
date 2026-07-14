package dev.ggid.sdk;

import java.util.List;

public class Webhook {
    public String id;
    public String url;
    public List<String> events;
    public String description;
    public boolean active;
    public String created_at;
    public String secret;

    public Webhook() {}

    public Webhook(String url, List<String> events, String description) {
        this.url = url;
        this.events = events;
        this.description = description;
        this.active = true;
    }
}
