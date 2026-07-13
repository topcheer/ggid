"""Tests for GGID Python SDK."""
import unittest
from unittest.mock import patch, MagicMock
from ggid.client import GGIDClient, GGIDConfig, GGIDError


class TestGGIDClient(unittest.TestCase):
    def setUp(self):
        self.config = GGIDConfig(base_url="http://localhost:8080")
        self.client = GGIDClient(self.config)

    def test_config_defaults(self):
        c = GGIDConfig()
        self.assertEqual(c.base_url, "http://localhost:8080")
        self.assertEqual(c.timeout, 30)

    def test_headers(self):
        h = self.client._headers()
        self.assertEqual(h["Content-Type"], "application/json")
        self.assertIn("X-Tenant-ID", h)

    def test_headers_with_token(self):
        h = self.client._headers(token="abc123")
        self.assertEqual(h["Authorization"], "Bearer abc123")

    def test_headers_with_api_key(self):
        self.config.api_key = "key-123"
        h = self.client._headers()
        self.assertEqual(h["X-API-Key"], "key-123")

    @patch("ggid.client.requests.Session.request")
    def test_register(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_resp.json.return_value = {"user_id": "123"}
        mock_resp.text = '{"user_id": "123"}'
        mock_req.return_value = mock_resp
        result = self.client.register("user1", "user1@test.com", "Pass123!")
        self.assertEqual(result["user_id"], "123")

    @patch("ggid.client.requests.Session.request")
    def test_login(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"access_token": "tok123"}
        mock_resp.text = '{"access_token": "tok123"}'
        mock_req.return_value = mock_resp
        result = self.client.login("user1", "Pass123!")
        self.assertEqual(result["access_token"], "tok123")

    @patch("ggid.client.requests.Session.request")
    def test_check_permission(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"allowed": True, "reason": "role:admin"}
        mock_resp.text = '{"allowed": true}'
        mock_req.return_value = mock_resp
        result = self.client.check_permission("tok", "users", "read")
        self.assertTrue(result["allowed"])

    @patch("ggid.client.requests.Session.request")
    def test_evaluate_abac(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"allowed": False, "reason": "denied by policy"}
        mock_resp.text = '{"allowed": false}'
        mock_req.return_value = mock_resp
        result = self.client.evaluate_abac(
            "tok", "read", "documents/secret", "user-123",
            conditions=[{"field": "department", "operator": "eq", "value": "finance"}],
        )
        self.assertFalse(result["allowed"])

    @patch("ggid.client.requests.Session.request")
    def test_create_role(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_resp.json.return_value = {"id": "role-1", "name": "admin"}
        mock_resp.text = '{"id": "role-1"}'
        mock_req.return_value = mock_resp
        result = self.client.create_role("tok", "admin", "admin", "Admin role")
        self.assertEqual(result["id"], "role-1")

    @patch("ggid.client.requests.Session.request")
    def test_assign_role(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"status": "assigned"}
        mock_resp.text = '{"status": "assigned"}'
        mock_req.return_value = mock_resp
        result = self.client.assign_role("tok", "user-1", "role-1")
        self.assertEqual(result["status"], "assigned")

    @patch("ggid.client.requests.Session.request")
    def test_api_error(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 404
        mock_resp.json.return_value = {"error": "not found"}
        mock_resp.text = '{"error": "not found"}'
        mock_req.return_value = mock_resp
        with self.assertRaises(GGIDError) as ctx:
            self.client.get_user("tok", "nonexistent")
        self.assertEqual(ctx.exception.status_code, 404)

    @patch("ggid.client.requests.Session.request")
    def test_get_oidc_discovery(self, mock_req):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.json.return_value = {"issuer": "http://localhost:9005"}
        mock_resp.text = '{"issuer": "http://localhost:9005"}'
        mock_req.return_value = mock_resp
        result = self.client.get_oidc_discovery()
        self.assertEqual(result["issuer"], "http://localhost:9005")


class TestJWTVerifier(unittest.TestCase):
    def test_jwt_error_no_kid(self):
        from ggid.jwt_verifier import JWTVerifier, JWTError
        import jwt as pyjwt
        # Create token without kid
        token = pyjwt.encode({"sub": "123"}, "secret", algorithm="HS256")
        v = JWTVerifier("http://localhost:8080")
        with self.assertRaises(JWTError):
            v.verify(token)


class TestMiddleware(unittest.TestCase):
    def test_get_token_from_header_valid(self):
        from ggid.middleware import _get_token_from_header
        token = _get_token_from_header("Bearer abc123")
        self.assertEqual(token, "abc123")

    def test_get_token_from_header_missing(self):
        from ggid.middleware import _get_token_from_header
        from ggid.jwt_verifier import JWTError
        with self.assertRaises(JWTError):
            _get_token_from_header("")

    def test_get_token_from_header_invalid_format(self):
        from ggid.middleware import _get_token_from_header
        from ggid.jwt_verifier import JWTError
        with self.assertRaises(JWTError):
            _get_token_from_header("Token abc123")


if __name__ == "__main__":
    unittest.main()
