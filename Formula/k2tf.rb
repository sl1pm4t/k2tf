class K2tf < Formula
  desc "Kubernetes YAML to Terraform HCL converter"
  homepage "https://github.com/sl1pm4t/k2tf"
  url      "https://github.com/sl1pm4t/k2tf/archive/v0.5.0.tar.gz"
  sha256   "94113ffb4874b9206148b7ea8bda56f30381b037547651e7c19af8547d845706"
  head     "https://github.com/sl1pm4t/k2tf.git"

  bottle :unneeded

  depends_on "go" => :build

  def install
    system "go", "build"
    bin.install "k2tf"
  end

  test do
    system "#{bin}/k2tf", "-v"
  end
end
