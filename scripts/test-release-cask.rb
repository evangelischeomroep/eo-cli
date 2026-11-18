require "cask/cask_loader"
require "simulate_system"

tag, cask_path, checksums_path = ARGV
abort "Usage: brew ruby -- scripts/test-release-cask.rb vX.Y.Z eo.rb checksums.txt" unless checksums_path

checksums = File.readlines(checksums_path).to_h do |line|
  digest, filename = line.split
  [filename, digest]
end

{ macos: "darwin", linux: "linux" }.each do |os, goos|
  { arm: "arm64", intel: "amd64" }.each do |arch, goarch|
    filename = "eo_#{goos}_#{goarch}.tar.gz"
    url = "https://github.com/evangelischeomroep/eo-cli/releases/download/#{tag}/#{filename}"
    Homebrew::SimulateSystem.with(os: os, arch: arch) do
      cask = Cask::CaskLoader::FromPathLoader.new(cask_path).load(config: nil)
      unless cask.version.to_s == tag.delete_prefix("v") && cask.url.to_s == url && cask.sha256.to_s == checksums.fetch(filename)
        abort "Wrong Cask version, URL or SHA-256 for #{os}/#{arch} (expected #{filename})"
      end
    end
  end
end

puts "All four Homebrew targets select the expected eo archive and SHA-256."
