package com.example.erp;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * ERP Demo Application — demonstrates GGID fine-grained RBAC.
 *
 * 6 roles, 8 modules, different read/write/approve permissions per role.
 * Shows how an external application integrates with GGID for authentication
 * and authorization.
 */
@SpringBootApplication
public class ErpDemoApplication {
    public static void main(String[] args) {
        SpringApplication.run(ErpDemoApplication.class, args);
    }
}
