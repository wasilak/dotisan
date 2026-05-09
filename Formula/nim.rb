class Nim < Formula
  desc "Declarative dotfiles management for developers"
  homepage "https://github.com/wasilak/nim"
  url "https://github.com/wasilak/nim/archive/refs/tags/v0.0.0.tar.gz"
  sha256 "TODO"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/wasilak/nim/cmd.Version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    # Minimal smoke test: version output
    assert_match version.to_s, shell_output("#{bin}/nim --version")
  end
end
