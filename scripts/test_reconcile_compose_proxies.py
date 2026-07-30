from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("reconcile_compose_proxies.py")
SPEC = importlib.util.spec_from_file_location("reconcile_compose_proxies", MODULE_PATH)
assert SPEC and SPEC.loader
reconciler = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = reconciler
SPEC.loader.exec_module(reconciler)


class ReconcileComposeTests(unittest.TestCase):
    endpoint = reconciler.ProxyEndpoint("172.18.0.1", 7891, "traefik", "172.18.0.0/16")

    def test_mapping_environment_preserves_unrelated_values(self) -> None:
        document = {
            "services": {
                "app": {
                    "networks": ["traefik"],
                    "environment": {"KEEP": "value", "HTTP_PROXY": "http://old:1"},
                    "extra_hosts": ["cache.internal:10.0.0.10"],
                    "labels": ["traefik.enable=true"],
                },
            },
        }

        changed = reconciler.reconcile_compose(document, self.endpoint, "host.docker.internal", [], None)

        service = document["services"]["app"]
        self.assertEqual(["app"], changed)
        self.assertEqual("value", service["environment"]["KEEP"])
        self.assertEqual("http://host.docker.internal:7891", service["environment"]["HTTP_PROXY"])
        self.assertIn("cache.internal:10.0.0.10", service["extra_hosts"])
        self.assertIn("host.docker.internal:172.18.0.1", service["extra_hosts"])
        self.assertEqual(["traefik.enable=true"], service["labels"])

    def test_list_environment_deduplicates_proxy_values(self) -> None:
        document = {
            "services": {
                "app": {
                    "networks": ["traefik"],
                    "environment": [
                        "PUID=1000",
                        "HTTP_PROXY=http://old:1",
                        "HTTP_PROXY=http://older:2",
                        "NO_PROXY=internal.example",
                    ],
                    "extra_hosts": ["host.docker.internal:1.1.1.1", "host.docker.internal:2.2.2.2"],
                },
            },
        }

        reconciler.reconcile_compose(document, self.endpoint, "host.docker.internal", ["custom.local"], None)

        environment = document["services"]["app"]["environment"]
        self.assertIn("PUID=1000", environment)
        self.assertEqual(1, sum(item.startswith("HTTP_PROXY=") for item in environment))
        no_proxy = next(item for item in environment if item.startswith("NO_PROXY="))
        self.assertIn("internal.example", no_proxy)
        self.assertIn("custom.local", no_proxy)
        self.assertIn("app", no_proxy)
        self.assertEqual(["host.docker.internal:172.18.0.1"], document["services"]["app"]["extra_hosts"])

    def test_second_reconciliation_is_idempotent(self) -> None:
        document = {"services": {"app": {"networks": ["traefik"]}}}

        first = reconciler.reconcile_compose(document, self.endpoint, "host.docker.internal", [], None)
        second = reconciler.reconcile_compose(document, self.endpoint, "host.docker.internal", [], None)

        self.assertEqual(["app"], first)
        self.assertEqual([], second)

    def test_network_filter_skips_other_services(self) -> None:
        document = {
            "services": {
                "target": {"networks": ["traefik"]},
                "other": {"networks": ["other-network"]},
            },
        }

        changed = reconciler.reconcile_compose(document, self.endpoint, "host.docker.internal", [], "traefik")

        self.assertEqual(["target"], changed)
        self.assertNotIn("environment", document["services"]["other"])

    def test_listener_parser_only_accepts_mihomo(self) -> None:
        output = "\n".join(
            [
                'LISTEN 0 4096 172.18.0.1:7891 0.0.0.0:* users:(("mihomo",pid=1,fd=3))',
                'LISTEN 0 4096 172.18.0.1:7892 0.0.0.0:* users:(("other",pid=2,fd=3))',
            ],
        )
        self.assertEqual({("172.18.0.1", 7891)}, reconciler.parse_mihomo_listeners(output))

    def test_invalid_environment_is_rejected(self) -> None:
        with self.assertRaises(reconciler.ReconcileError):
            reconciler.environment_as_mapping("HTTP_PROXY=http://proxy")


if __name__ == "__main__":
    unittest.main()
