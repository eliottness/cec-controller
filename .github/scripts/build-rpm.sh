#!/bin/bash
set -e

ARCH=$1
VERSION=$2
BINARY_PATH=$3

if [ -z "$ARCH" ] || [ -z "$VERSION" ] || [ -z "$BINARY_PATH" ]; then
  echo "Usage: $0 <arch> <version> <binary_path>"
  exit 1
fi

# Install rpm-build tools
dnf install -y rpm-build rpmdevtools

# Setup rpmbuild directories
mkdir -p /root/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
mkdir -p "/root/rpmbuild/BUILD/cec-controller-${VERSION}/usr/bin"

# Copy binary
cp "$BINARY_PATH" "/root/rpmbuild/BUILD/cec-controller-${VERSION}/usr/bin/"

# Create spec file
cat > /root/rpmbuild/SPECS/cec-controller.spec <<EOF
Name:           cec-controller
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        CEC Controller Service for Fedora/RHEL
License:        GPLv3
URL:            https://github.com/eliottness/cec-controller
BuildArch:      ${ARCH}

%description
CEC Controller Service for Fedora/RHEL

%install
mkdir -p %{buildroot}/usr/bin
cp %{_builddir}/cec-controller-%{version}/usr/bin/cec-controller %{buildroot}/usr/bin/

%files
/usr/bin/cec-controller

%changelog
* $(date +'%a %b %d %Y') eliottness - ${VERSION}-1
- Release ${VERSION}
EOF

# Build RPM
rpmbuild -bb /root/rpmbuild/SPECS/cec-controller.spec

# Find and copy the generated RPM
if [ "$ARCH" = "x86_64" ]; then
  RPM_DIR="/root/rpmbuild/RPMS/x86_64"
elif [ "$ARCH" = "aarch64" ]; then
  RPM_DIR="/root/rpmbuild/RPMS/aarch64"
else
  echo "Unknown architecture: $ARCH"
  exit 1
fi

cp ${RPM_DIR}/*.rpm /workspace/
echo "RPM created successfully:"
ls -la /workspace/*.rpm
