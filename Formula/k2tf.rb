class K2tf < Formula
  desc "Kubernetes YAML to Terraform HCL converter"
  homepage "https://github.com/sl1pm4t/k2tf"
  url      "https://github.com/sl1pm4t/k2tf/archive/v0.4.1.tar.gz"
  sha256   "d6070de2afc7bacf8a6ead24ad81b7d94f16e5106941ac78ced9b795ff4f8403"
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
