"""Tests for template rendering in ApplicationManager."""

from unittest.mock import patch

import pytest

from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl


@pytest.fixture
def app_manager():
    """Create ApplicationManager instance."""
    from pomelo_orbit.infrastructure.config import get_settings

    return ApplicationManagerImpl(get_settings())


class TestRenderTemplate:
    """Tests for _render_template method."""

    def test_renders_jinja2_template(self, app_manager):
        """Should render Jinja template content."""
        content = "dir: {{ app.physical_dir }}"
        rendered = app_manager._render_template("test-app", content)
        assert "dir:" in rendered
        assert rendered != content

    def test_provides_settings_in_context(self, app_manager):
        """Should provide app.physical_dir and app.physical_app_dir in template context."""
        content = "dir: {{ app.physical_dir }}, app_dir: {{ app.physical_app_dir }}"
        rendered = app_manager._render_template("test-app", content)
        assert rendered != content
        assert "{{ app.physical_dir }}" not in rendered
        assert "{{ app.physical_app_dir }}" not in rendered

    def test_provides_config_in_context(self, app_manager):
        """Should provide config.domain_suffix in template context."""
        content = "domain: {{ config.domain_suffix }}"
        rendered = app_manager._render_template("test-app", content)
        assert "domain:" in rendered
        assert "{{ config.domain_suffix }}" not in rendered

    def test_supports_jinja2_conditionals(self, app_manager):
        """Should support Jinja2 conditional statements."""
        content = "{% if cert.letsencrypt.enabled %}ssl: true{% else %}ssl: false{% endif %}"
        rendered = app_manager._render_template("test-app", content)
        assert "ssl:" in rendered

    def test_raises_error_on_template_syntax_error(self, app_manager):
        """Should raise ValueError on template syntax error."""
        content = "{{ unclosed"
        with pytest.raises(ValueError, match="Template rendering failed"):
            app_manager._render_template("test-app", content)


class TestWriteFileWithTemplate:
    """Tests for write_file method with template support."""

    def test_writes_non_template_file_directly(self, app_manager, tmp_path):
        """Should write non-template file directly."""
        test_dir = tmp_path / "test-app"
        test_dir.mkdir(parents=True, exist_ok=True)
        with patch.object(app_manager, "get_app_working_dir", return_value=test_dir):
            app_manager._write_file("test-app", "test.txt", "plain content")
            assert (test_dir / "test.txt").read_text() == "plain content"

    def test_renders_and_writes_template_file(self, app_manager, tmp_path):
        """Should render template and write to file without .jinja extension."""
        content = "value: {{ app.physical_app_dir }}"
        test_dir = tmp_path / "render-test"
        test_dir.mkdir(parents=True, exist_ok=True)
        with patch.object(app_manager, "get_app_working_dir", return_value=test_dir):
            app_manager._write_file("test-app", "config.yml.jinja", content)

        assert not (test_dir / "config.yml.jinja").exists()
        assert (test_dir / "config.yml").exists()
        rendered_content = (test_dir / "config.yml").read_text()
        assert rendered_content != "value: {{ app.physical_app_dir }}"

    def test_creates_parent_directories_for_template(self, app_manager, tmp_path):
        """Should create parent directories for template files."""
        content = "test: {{ app.physical_app_dir }}"
        test_dir = tmp_path / "parent-test"
        with patch.object(app_manager, "get_app_working_dir", return_value=test_dir):
            app_manager._write_file("test-app", "data/config.yml.jinja", content)

        assert (test_dir / "data" / "config.yml").exists()

    def test_sets_file_mode_for_init_sh(self, app_manager, tmp_path):
        """Should set executable mode for init.sh."""
        content = "#!/bin/bash\necho {{ app.physical_app_dir }}"
        test_dir = tmp_path / "init-test"
        test_dir.mkdir(parents=True, exist_ok=True)
        with patch.object(app_manager, "get_app_working_dir", return_value=test_dir):
            app_manager._write_file("test-app", "init.sh.jinja", content)

        file_path = test_dir / "init.sh"
        assert file_path.exists()
        assert file_path.stat().st_mode & 0o755
