# frozen_string_literal: true

require_relative "lib/ggid/version"

Gem::Specification.new do |spec|
  spec.name = "ggid"
  spec.version = GGID::VERSION
  spec.summary = "GGID IAM Platform Ruby SDK"
  spec.description = "JWT verification, OAuth/OIDC, RBAC, ABAC, and Rails middleware for the GGID IAM Platform"
  spec.authors = ["GGID Team"]
  spec.email = ["noreply@ggid.dev"]
  spec.license = "Apache-2.0"
  spec.homepage = "https://github.com/ggid/ggid"
  spec.required_ruby_version = ">= 2.6"

  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = "https://github.com/ggid/ggid/tree/main/sdk/ruby"

  # Dependencies
  spec.add_dependency "httparty", ">= 0.20", "< 0.22"
  spec.add_dependency "jwt", "~> 2.8"

  # Development dependencies
  spec.add_development_dependency "rspec", "~> 3.13"
  spec.add_development_dependency "webmock", "~> 3.23"
  spec.add_development_dependency "rake", "~> 13.0"

  spec.files = Dir.chdir(__dir__) do
    Dir.glob("{lib,examples}/**/*") + ["README.md", "ggid.gemspec"]
  end
  spec.require_paths = ["lib"]
end
