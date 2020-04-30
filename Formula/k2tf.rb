class K2tf < Formula
  desc "Kubernetes YAML to Terraform HCL converter"
  homepage "https://github.com/sl1pm4t/k2tf"
  url      "https://github.com/sl1pm4t/k2tf/archive/v0.4.1.tar.gz"
  sha256   "475478b3d8bb7b5af8201b3e512b78548875d93d88883bed1f52fc3e00bfac11"
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
