FROM mcr.microsoft.com/dotnet/sdk:8.0
WORKDIR /app
COPY oauth-demo.csproj .
RUN dotnet build 2>/dev/null || true
COPY . .
ENV PORT=8080
EXPOSE 8080
CMD ["dotnet", "run"]
