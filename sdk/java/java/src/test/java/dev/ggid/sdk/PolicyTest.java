package dev.ggid.sdk;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for RBAC/ABAC SDK model classes and serialization.
 */
class PolicyTest {

    private static final ObjectMapper mapper = new ObjectMapper();

    @Test
    void testPolicyResultSerialization() throws Exception {
        PolicyResult result = new PolicyResult(true, "allowed by admin role");
        String json = mapper.writeValueAsString(result);
        assertTrue(json.contains("\"allowed\":true"));
        assertTrue(json.contains("\"allowed by admin role\""));

        PolicyResult deserialized = mapper.readValue(json, PolicyResult.class);
        assertTrue(deserialized.isAllowed());
        assertEquals("allowed by admin role", deserialized.getReason());
    }

    @Test
    void testPolicyResultDeserialization() throws Exception {
        String json = "{\"allowed\":false,\"reason\":\"denied: insufficient permissions\"}";
        PolicyResult result = mapper.readValue(json, PolicyResult.class);
        assertFalse(result.isAllowed());
        assertEquals("denied: insufficient permissions", result.getReason());
    }

    @Test
    void testPolicyCheckRequestSerialization() throws Exception {
        PolicyCheckRequest req = new PolicyCheckRequest(
            "user-123",
            "document:456",
            "read",
            Map.of("department", "engineering", "time", "2025-01-15T10:00:00Z"),
            "tenant-001"
        );
        String json = mapper.writeValueAsString(req);
        assertTrue(json.contains("\"subject\":\"user-123\""));
        assertTrue(json.contains("\"resource\":\"document:456\""));
        assertTrue(json.contains("\"action\":\"read\""));
        assertTrue(json.contains("\"tenant_id\":\"tenant-001\""));
        assertTrue(json.contains("engineering"));
    }

    @Test
    void testPolicyCheckRequestDeserialization() throws Exception {
        String json = "{\"subject\":\"alice\",\"resource\":\"report\",\"action\":\"write\",\"context\":{\"role\":\"manager\"},\"tenant_id\":\"t1\"}";
        PolicyCheckRequest req = mapper.readValue(json, PolicyCheckRequest.class);
        assertEquals("alice", req.getSubject());
        assertEquals("report", req.getResource());
        assertEquals("write", req.getAction());
        assertEquals("manager", req.getContext().get("role"));
        assertEquals("t1", req.getTenantId());
    }

    @Test
    void testABACConditionSerialization() throws Exception {
        ABACCondition cond = new ABACCondition("department", "eq", "engineering");
        String json = mapper.writeValueAsString(cond);
        assertTrue(json.contains("\"field\":\"department\""));
        assertTrue(json.contains("\"operator\":\"eq\""));
        assertTrue(json.contains("\"value\":\"engineering\""));
    }

    @Test
    void testABACConditionDeserialization() throws Exception {
        String json = "{\"field\":\"ip_address\",\"operator\":\"startsWith\",\"value\":\"192.168.\"}";
        ABACCondition cond = mapper.readValue(json, ABACCondition.class);
        assertEquals("ip_address", cond.getField());
        assertEquals("startsWith", cond.getOperator());
        assertEquals("192.168.", cond.getValue());
    }

    @Test
    void testABACEvalRequestSerialization() throws Exception {
        ABACEvalRequest req = new ABACEvalRequest(
            Map.of("role", "admin", "department", "eng"),
            List.of(
                new ABACCondition("role", "eq", "admin"),
                new ABACCondition("department", "in", "eng,sales")
            )
        );
        String json = mapper.writeValueAsString(req);
        assertTrue(json.contains("\"attributes\""));
        assertTrue(json.contains("\"conditions\""));
        assertTrue(json.contains("\"role\""));
        assertTrue(json.contains("\"admin\""));
    }

    @Test
    void testABACEvalResultDeserialization() throws Exception {
        String json = "{\"matched\":true,\"matched_rules\":[\"rule-1\",\"rule-3\"]}";
        ABACEvalResult result = mapper.readValue(json, ABACEvalResult.class);
        assertTrue(result.isMatched());
        assertEquals(2, result.getMatchedRules().size());
        assertEquals("rule-1", result.getMatchedRules().get(0));
        assertEquals("rule-3", result.getMatchedRules().get(1));
    }

    @Test
    void testABACEvalResultNoMatchedRules() throws Exception {
        String json = "{\"matched\":false}";
        ABACEvalResult result = mapper.readValue(json, ABACEvalResult.class);
        assertFalse(result.isMatched());
        assertNull(result.getMatchedRules());
    }

    @Test
    void testPermissionDeserialization() throws Exception {
        String json = "{\"id\":\"perm-1\",\"name\":\"View Users\",\"resource\":\"users\",\"action\":\"read\",\"description\":\"View user profiles\",\"children\":[]}";
        Permission perm = mapper.readValue(json, Permission.class);
        assertEquals("perm-1", perm.getId());
        assertEquals("View Users", perm.getName());
        assertEquals("users", perm.getResource());
        assertEquals("read", perm.getAction());
        assertEquals("View user profiles", perm.getDescription());
        assertNotNull(perm.getChildren());
        assertTrue(perm.getChildren().isEmpty());
    }

    @Test
    void testPermissionWithChildren() throws Exception {
        String json = "{\"id\":\"root\",\"name\":\"All\",\"resource\":\"*\",\"action\":\"*\",\"children\":[{\"id\":\"c1\",\"name\":\"Read\",\"resource\":\"docs\",\"action\":\"read\",\"children\":null}]}";
        Permission perm = mapper.readValue(json, Permission.class);
        assertEquals(1, perm.getChildren().size());
        assertEquals("Read", perm.getChildren().get(0).getName());
        assertEquals("docs", perm.getChildren().get(0).getResource());
    }

    @Test
    void testRoleDeserialization() throws Exception {
        String json = "{\"id\":\"role-1\",\"key\":\"admin\",\"name\":\"Administrator\",\"description\":\"Full access\",\"system_role\":true}";
        Role role = mapper.readValue(json, Role.class);
        assertEquals("role-1", role.getId());
        assertEquals("admin", role.getKey());
        assertEquals("Administrator", role.getName());
        assertEquals("Full access", role.getDescription());
        assertTrue(role.systemRole);
    }

    @Test
    void testRoleListDeserialization() throws Exception {
        String json = "[{\"id\":\"r1\",\"key\":\"viewer\",\"name\":\"Viewer\",\"system_role\":false},{\"id\":\"r2\",\"key\":\"editor\",\"name\":\"Editor\",\"system_role\":false}]";
        List<Role> roles = mapper.readValue(json, new com.fasterxml.jackson.core.type.TypeReference<List<Role>>() {});
        assertEquals(2, roles.size());
        assertEquals("viewer", roles.get(0).getKey());
        assertEquals("editor", roles.get(1).getKey());
    }

    @Test
    void testPolicyResultWithExtraFieldsIgnored() throws Exception {
        String json = "{\"allowed\":true,\"reason\":\"ok\",\"extra_field\":\"ignored\",\"trace_id\":\"abc\"}";
        PolicyResult result = mapper.readValue(json, PolicyResult.class);
        assertTrue(result.isAllowed());
        assertEquals("ok", result.getReason());
    }

    @Test
    void testABACConditionAllOperators() {
        String[] operators = {"eq", "ne", "in", "regex", "startsWith", "endsWith", "gt", "lt"};
        for (String op : operators) {
            ABACCondition cond = new ABACCondition("field", op, "value");
            assertEquals(op, cond.getOperator());
        }
    }

    @Test
    void testPolicyCheckRequestMinimalConstructor() {
        PolicyCheckRequest req = new PolicyCheckRequest("user1", "res1", "act1");
        assertEquals("user1", req.getSubject());
        assertEquals("res1", req.getResource());
        assertEquals("act1", req.getAction());
        assertNull(req.getContext());
        assertNull(req.getTenantId());
    }
}
