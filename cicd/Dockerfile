FROM golang:1.24.1-bookworm

# Imutable

RUN mkdir /dnsactioneer

WORKDIR /dnsactioneer

ENTRYPOINT ./dnsactioneer

# Mutable
COPY dnsactioneer dnsactioneer