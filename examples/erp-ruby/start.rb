# Startup wrapper: load app class, ensure HostAuthorization is removed, then run
require_relative 'app.rb'

# Belt and suspenders: even with set :protection, except: :host_authorization,
# forcefully strip any HostAuthorization from the middleware stack
ERPApp.middleware.delete_if do |m|
  m[0].to_s.include?('HostAuthorization')
end

ERPApp.run!
