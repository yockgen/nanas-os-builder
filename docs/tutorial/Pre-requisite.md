# General Build Prerequisites for Image Creation Tools

This document provides a list of general dependencies and steps required for installing them.

---
## Ukify
1.Prerequisites

Install all required dependencies:

```bash
sudo apt install git python3 python3-cryptography python3-pefile python3-pillow \
  python3-setuptools libssl-dev libzstd-dev uuid-dev gnu-efi python3-packaging \
  libelf-dev lz4 pkg-config meson ninja-build
```

2.Clone systemd Repository (for ukify)

Download the systemd source code and checkout the desired version:

```bash
git clone https://github.com/systemd/systemd.git
cd systemd
git checkout v255
```

3.Install `ukify`

Copy the `ukify.py` script to a directory in your PATH:

```bash
cd src/ukify
sudo cp ukify.py /usr/local/bin/ukify
```

4.Copy to `/usr/bin`

For environments that require `ukify` in `/usr/bin` (e.g., ICT build systems):

```bash
sudo cp /usr/local/bin/ukify /usr/bin/ukify
```

5.Verify Installation

Run the following to ensure `ukify` is correctly installed and accessible:

```bash
ukify --help
```

You should see the usage instructions for `ukify`.

---

