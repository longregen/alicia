(import (
  fetchTarball {
    url = "https://github.com/edolstra/flake-compat/archive/master.tar.gz";
    sha256 = "1c5f7vfn205bj4bmkgzgyw9504xh5j7gcwi8jf7yh581bwzlwl71";
  }
) {
  src = ./.;
}).shellNix.default
