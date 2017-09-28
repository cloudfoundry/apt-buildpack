#!/usr/bin/env ruby
require "webrick"

server = WEBrick::HTTPServer.new(:Port => (ENV['PORT'] || 8080))
server.mount_proc '/' do |req, res|
  res.body = "Ascii: #{`ascii d`}"
end

trap("INT") { server.shutdown }
server.start
