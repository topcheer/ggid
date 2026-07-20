FROM dart:stable
WORKDIR /app
COPY pubspec.yaml .
COPY oauth_demo.dart .
ENV PORT=8080
EXPOSE 8080
CMD ["dart", "run", "oauth_demo.dart"]
