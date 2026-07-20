FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /app

# Copy SDK and demo
COPY sdk/csharp/ ./sdk/
COPY examples/erp-csharp/ ./erp/

# Restore and build SDK first
WORKDIR /app/sdk
RUN dotnet restore

# Build demo
WORKDIR /app/erp
RUN dotnet restore && dotnet publish -c Release -o /out

FROM mcr.microsoft.com/dotnet/runtime:9.0
WORKDIR /app
COPY --from=build /out ./
EXPOSE 9200
CMD ["dotnet", "erp-csharp.dll"]
