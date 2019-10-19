Name: api-server
Version: 0.0.1
Release: 1
Summary: API Server
License: FIXME

# disable facist builds, we dont care about files we arent installing
%define _unpackaged_files_terminate_build 0

%description
a bad description for an awesome package

%prep

%build
go build -o api-server

%install
mkdir -p %{buildroot}/usr/local/bin
mkdir -p %{buildroot}/etc/systemd/system
install -m 755 api-server %{buildroot}/usr/local/bin/api-server
install -m 755 api-server.service %{buildroot}/etc/systemd/system/api-server.service

%files
/usr/local/bin/api-server
/etc/systemd/system/api-server.service

%changelog
# We will revisit
