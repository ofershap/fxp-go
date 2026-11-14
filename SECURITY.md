# Security Policy

## Reporting a vulnerability

Please report security issues through GitHub's private vulnerability reporting
("Report a vulnerability" on the Security tab of this repository). Do not open
a public issue for a vulnerability.

## Scope notes

fxp-go parses untrusted XML: it never fetches external resources (no network
code), never re-expands entities, and can cap nesting depth with
MaxNestedTags. There are no built-in input-size caps; bound input size in
your own code when parsing untrusted documents.
