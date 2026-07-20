using System;
using System.Net.Http;
using System.Xml.Linq;

namespace GGID
{
    /// <summary>
    /// GGID SAML SDK for C#.
    /// Generates SP metadata for IdP integration.
    /// </summary>
    public class SAML
    {
        public string EntityId { get; }
        public string AcsUrl { get; }
        public string SloUrl { get; }

        /// <param name="entityId">SP Entity ID (e.g. https://myapp.example.com/saml)</param>
        /// <param name="acsUrl">Assertion Consumer Service URL</param>
        /// <param name="sloUrl">Single Logout URL (optional)</param>
        public SAML(string entityId, string acsUrl, string sloUrl = "")
        {
            if (string.IsNullOrEmpty(entityId))
                throw new ArgumentException("entityId is required");
            if (string.IsNullOrEmpty(acsUrl))
                throw new ArgumentException("acsUrl is required");

            EntityId = entityId;
            AcsUrl = acsUrl;
            SloUrl = sloUrl;
        }

        /// <summary>
        /// Generate SP metadata XML string.
        /// </summary>
        public string GenerateSPMetadata()
        {
            var ns = XNamespace.Get("urn:oasis:names:tc:SAML:2.0:metadata");
            var validUntil = DateTime.UtcNow.AddYears(1).ToString("o");

            var doc = new XDocument(
                new XElement(ns + "EntityDescriptor",
                    new XAttribute("entityID", EntityId),
                    new XAttribute("validUntil", validUntil),
                    new XElement(ns + "SPSSODescriptor",
                        new XAttribute("protocolSupportEnumeration", "urn:oasis:names:tc:SAML:2.0:protocol"),
                        new XElement(ns + "NameIDFormat", "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"),
                        new XElement(ns + "AssertionConsumerService",
                            new XAttribute("Binding", "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"),
                            new XAttribute("Location", AcsUrl),
                            new XAttribute("index", 0)
                        )
                    )
                )
            );

            // Add SLO if configured
            if (!string.IsNullOrEmpty(SloUrl))
            {
                var spDescriptor = doc.Root.Element(ns + "SPSSODescriptor");
                spDescriptor.Add(new XElement(ns + "SingleLogoutService",
                    new XAttribute("Binding", "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"),
                    new XAttribute("Location", SloUrl)
                ));
            }

            return doc.Declaration + "\n" + doc.ToString();
        }

        /// <summary>
        /// Fetch IdP metadata from GGID instance.
        /// </summary>
        /// <param name="ggidBaseUrl">e.g. https://ggid.example.com</param>
        public static string FetchIdPMetadata(string ggidBaseUrl)
        {
            var url = ggidBaseUrl.TrimEnd('/') + "/saml/idp/metadata";
            using var client = new HttpClient();
            return client.GetStringAsync(url).GetAwaiter().GetResult();
        }
    }
}