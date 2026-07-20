FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /src
COPY examples/erp-csharp/ ./erp/
WORKDIR /src/erp
RUN dotnet publish -c Release -o /out

FROM mcr.microsoft.com/dotnet/runtime:9.0
WORKDIR /app
COPY --from=build /out ./
EXPOSE 9200
CMD ["dotnet", "erp-csharp.dll"]
