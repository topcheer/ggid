FROM ruby:3.3-alpine
RUN apk add --no-cache ca-certificates build-base && \
    gem install sinatra jwt httparty rackup puma
COPY sdk/ruby/ /app/sdk/ruby/
COPY examples/erp-ruby/ /app/examples/erp-ruby/
WORKDIR /app/examples/erp-ruby
ENV ERP_LISTEN=9091
EXPOSE 9091
CMD ["ruby", "start.rb"]
