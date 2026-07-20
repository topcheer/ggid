"""SAML Service Provider configuration and metadata generation.

Usage::

    from ggid.saml import SAMLConfig, generate_sp_metadata

    config = SAMLConfig(
        entity_id="https://myapp.example.com/saml",
        acs_url="https://myapp.example.com/saml/acs",
        slo_url="https://myapp.example.com/saml/slo",
    )
    metadata_xml = generate_sp_metadata(config)
"""

from __future__ import annotations
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from xml.sax.saxutils import escape


@dataclass
class SAMLConfig:
    """SAML Service Provider configuration."""
    entity_id: str
    acs_url: str
    slo_url: str = ""
    sign_requests: bool = False


def generate_sp_metadata(config: SAMLConfig) -> str:
    """Generate SAML SP metadata XML (EntityDescriptor).

    Args:
        config: SAMLConfig with entity_id and acs_url set.

    Returns:
        SP EntityDescriptor XML string.

    Raises:
        ValueError: If entity_id or acs_url is empty.
    """
    if not config.entity_id:
        raise ValueError("entity_id is required")
    if not config.acs_url:
        raise ValueError("acs_url is required")

    valid_until = (datetime.now(timezone.utc) + timedelta(days=365)).strftime(
        "%Y-%m-%dT%H:%M:%SZ"
    )

    parts = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        f'<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"',
        f' entityID="{escape(config.entity_id)}"',
        f' validUntil="{valid_until}">',
        '<SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">',
        "<NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>",
        '<AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"',
        f' Location="{escape(config.acs_url)}" index="0"/>',
    ]

    if config.slo_url:
        parts.append(
            '<SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"'
            f' Location="{escape(config.slo_url)}"/>'
        )

    parts.append("</SPSSODescriptor>")
    parts.append("</EntityDescriptor>")
    return "".join(parts)


def fetch_idp_metadata(client) -> bytes:
    """Fetch IdP metadata XML from a GGID instance.

    Args:
        client: A ggid.Client instance with a configured base_url.

    Returns:
        IdP metadata XML bytes.
    """
    import urllib.request
    url = f"{client.base_url}/saml/idp/metadata"
    with urllib.request.urlopen(url) as resp:
        return resp.read()
