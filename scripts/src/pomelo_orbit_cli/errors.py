"""Configuration errors rendered by the CLI boundary."""


class ConfigurationError(RuntimeError):
    """Raised when shared Orbit configuration cannot initialize safely."""
