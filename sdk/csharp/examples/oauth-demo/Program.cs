// GGID OAuth 2.0 Demo (C#)
// Run: GGID_URL=https://ggid.example.com CLIENT_ID=xxx dotnet run
using System;
using System.Net;
using System.Net.Http;
using System.Text.Json;
using System.Web;

class OAuthDemo
{
    static async Task Main()
    {
        var ggidUrl = Environment.GetEnvironmentVariable("GGID_URL") ?? "http://localhost:8080";
        var clientId = Environment.GetEnvironmentVariable("CLIENT_ID") ?? "";
        var clientSecret = Environment.GetEnvironmentVariable("CLIENT_SECRET") ?? "";
        var redirectUri = Environment.GetEnvironmentVariable("REDIRECT_URI") ?? "http://localhost:3000/callback";

        using var listener = new HttpListener();
        listener.Prefixes.Add("http://localhost:3000/");
        listener.Start();
        Console.WriteLine("OAuth demo on http://localhost:3000");

        using var http = new HttpClient();

        while (true)
        {
            var ctx = await listener.GetContextAsync();
            var path = ctx.Request.Url.AbsolutePath;

            if (path == "/")
            {
                var authUrl = $"{ggidUrl}/api/v1/oauth/authorize?response_type=code&client_id={clientId}" +
                    $"&redirect_uri={HttpUtility.UrlEncode(redirectUri)}&scope=openid+profile+email&state=demo";
                var html = $"<h1>GGID OAuth Demo</h1><a href='{authUrl}'>Login with GGID</a>";
                ctx.Response.ContentType = "text/html";
                var buf = System.Text.Encoding.UTF8.GetBytes(html);
                ctx.Response.OutputStream.Write(buf, 0, buf.Length);
                ctx.Response.Close();
            }
            else if (path == "/callback")
            {
                var code = ctx.Request.QueryString["code"];
                var tokenContent = new FormUrlEncodedContent(new Dictionary<string, string>
                {
                    ["grant_type"] = "authorization_code",
                    ["code"] = code,
                    ["redirect_uri"] = redirectUri,
                    ["client_id"] = clientId,
                    ["client_secret"] = clientSecret,
                });
                var tokenRes = await http.PostAsync($"{ggidUrl}/api/v1/oauth/token", tokenContent);
                var tokens = await tokenRes.Content.ReadAsStringAsync();

                ctx.Response.ContentType = "application/json";
                var buf = System.Text.Encoding.UTF8.GetBytes(tokens);
                ctx.Response.OutputStream.Write(buf, 0, buf.Length);
                ctx.Response.Close();
            }
        }
    }
}
