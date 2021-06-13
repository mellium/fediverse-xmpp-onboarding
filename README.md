# XMPP Self Provisioning with Mastodon


The `fediverse-xmpp-onboarding` tool is an example of a [new ProtoXEP for for
pre-auth key generation][proto] (it has not yet been accepted by the XMPP
Standards Foundation and is unofficial) that uses an existing Mastodon (or other
compatible fediverse tool) account to authorize a user to self-provision an XMPP
account using [XEP-0401: Easy User Onboarding] without requiring a connection to
the XMPP server.

No actual servers implement this ProtoXEP yet.


## License

The package may be used under the terms of the BSD 2-Clause License a copy of
which may be found in the file "[LICENSE]".


[proto]: https://github.com/xsf/xeps/pull/1068
[XEP-0401: Easy User Onboarding]: https://xmpp.org/extensions/xep-0401.html
[LICENSE]: https://github.com/mellium/fediverse-xmpp-onboarding/blob/main/LICENSE
