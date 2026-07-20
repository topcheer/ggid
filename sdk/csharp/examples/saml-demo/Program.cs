// GGID SAML SSO Demo (C#)
// Run: GGID_URL=https://ggid.example.com SP_ENTITY_ID=... dotnet run
using System;
using System.Net;
using System.Text;
using GGID;

class SamlDemo
{
    static async Task Main()
    {
        var ggidUrl = Environment.GetEnvironmentVariable("GGID_URL") ?? "http://localhost:8080";
        var entityId = Environment.GetEnvironmentVariable("SP_ENTITY_ID") ?? "http://localhost:3001/saml/metadata";
        var acsUrl = Environment.GetEnvironmentVariable("ACS_URL") ?? "http://localhost:3001/saml/acs";

        using var listener = new HttpListener();
        listener.Prefixes.Add("http://localhost:3001/");
        listener.Start();
        Console.WriteLine("SAML demo on http://localhost:3001");

        while (true)
        {
            var ctx = await listener.GetContextAsync();
            var path = ctx.Request.Url.AbsolutePath;
            var res = ctx.Response;

            if (path == "/")
            {
                res.ContentType = "text/html";
                var buf = Encoding.UTF8.GetBytes("<h1>GGID SAML Demo</h1><a href='/login'>Login</a>");
                res.OutputStream.Write(buf, 0, buf.Length);
            }
            else if (path == "/saml/metadata")
            {
                res.ContentType = "application/xml";
                var xml = SAML.GenerateSPMetadata(entityId, acsUrl);
                var buf = Encoding.UTF8.GetBytes(xml);
                res.OutputStream.Write(buf, 0, buf.Length);
            }
            else if (path == "/login")
            {
                res.Redirect($"{ggidUrl}/saml/sso?SPEntityId={Uri.EscapeDataString(entityId)}&ACSUrl={Uri.EscapeDataString(acsUrl)}");
            }
            else if (path == "/saml/acs" && ctx.Request.HttpMethod == "POST")
            {
                res.ContentType = "text/html";
                var buf = Encoding.UTF8.GetBytes("<h1>SAML ACS</h1><p>Received SAML response</p><a href='/profile'>Continue</a>");
                res.OutputStream.Write(buf, 0, buf.Length);
            }
            else if (path == "/profile")
            {
                res.ContentType = "text/html";
                var buf = Encoding.UTF8.GetBytes("<h1>Profile</h1><p>Authenticated via SAML</p>");
                res.OutputStream.Write(buf, 0, buf.Length);
            }
            res.Close();
        }
    }
}
