#!/usr/bin/env ruby
require "webrick"

server = WEBrick::HTTPServer.new(:Port => (ENV['PORT'] || 8080))
server.mount_proc '/jq' do |req, res|
  res.body = "Jq: #{`jq --version`}"
end

server.mount_proc '/bosh' do |req, res|
  res.body = "BOSH: #{`bosh2 -v`}"
end

trap("INT") { server.shutdown }
server.start
