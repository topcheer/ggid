FROM eclipse-temurin:21-jdk-alpine
WORKDIR /app
RUN wget -q https://repo1.maven.org/maven2/com/google/code/gson/gson/2.11.0/gson-2.11.0.jar -O gson.jar
COPY . .
RUN javac -cp gson.jar OAuthDemo.java
ENV PORT=8080
EXPOSE 8080
CMD ["java", "-cp", "gson.jar:.", "OAuthDemo"]
