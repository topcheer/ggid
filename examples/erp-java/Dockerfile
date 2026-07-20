FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /app

# Copy SDK
COPY sdk/java/ ./sdk/java/
RUN cd sdk/java && mvn install -DskipTests

# Copy demo (SDK is now in local maven repo)
COPY examples/erp-java/ ./erp/
WORKDIR /app/erp
RUN mvn package -DskipTests

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /app/erp/target/erp-java-demo-1.0.0.jar app.jar
EXPOSE 9300
CMD ["java", "-jar", "app.jar"]
