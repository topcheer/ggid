// SAML SSO Demo — Java SDK
import com.sun.net.httpserver.*;
import java.io.*;
import java.net.*;

public class SAMLSDemo {
    static String ggidUrl = System.getenv().getOrDefault("GGID_URL", "http://localhost:8080");
    static int port = Integer.parseInt(System.getenv().getOrDefault("PORT", "9095"));

    public static void main(String[] args) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/", ex -> {
            String email = ex.getRequestURI().getQuery() != null && ex.getRequestURI().getQuery().contains("email=") ? "user" : "";
            String html = email.isEmpty()
                ? "<h1>SAML SSO Demo</h1><a href='/saml/sso'>Login via SAML</a>"
                : "<h1>SAML SSO Demo</h1><p>Authenticated!</p>";
            ex.sendResponseHeaders(200, html.length());
            ex.getResponseBody().write(html.getBytes());
            ex.getResponseBody().close();
        });
        server.createContext("/saml/metadata", ex -> {
            String xml = "<?xml version=\"1.0\"?><EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"saml-demo\"><SPSSODescriptor protocolSupportEnumeration=\"urn:oasis:names:tc:SAML:2.0:protocol\"><NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat><AssertionConsumerService index=\"0\" isDefault=\"true\" Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST\" Location=\"http://localhost:" + port + "/saml/acs\"/></SPSSODescriptor></EntityDescriptor>";
            ex.getResponseHeaders().set("Content-Type", "application/xml");
            ex.sendResponseHeaders(200, xml.length());
            ex.getResponseBody().write(xml.getBytes());
            ex.getResponseBody().close();
        });
        server.createContext("/saml/sso", ex -> {
            ex.sendResponseHeaders(302, 0);
            ex.getResponseHeaders().set("Location", ggidUrl + "/saml/sso?RelayState=http://localhost:" + port + "/");
            ex.getResponseBody().close();
        });
        server.createContext("/saml/acs", ex -> {
            ex.sendResponseHeaders(302, 0);
            ex.getResponseHeaders().set("Location", "/?email=authenticated");
            ex.getResponseBody().close();
        });
        server.start();
        System.out.println("SAML demo on :" + port);
    }
}
