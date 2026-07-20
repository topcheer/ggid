FROM ruby:3.3-slim
WORKDIR /app
RUN gem install sinatra webrick
COPY . .
EXPOSE 4567
CMD ["ruby", "oauth_demo.rb"]
