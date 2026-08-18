"""Focused tests for the RAGFlow TEI preparation script."""

from __future__ import annotations

import argparse
from contextlib import redirect_stderr
import io
import json
import tarfile
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import prepare_ragflow_tei as preparation


class PreparationArchiveTests(unittest.TestCase):
    def create_model(self, root: Path) -> Path:
        model_dir = root / "bge-m3"
        (model_dir / "onnx").mkdir(parents=True)
        (model_dir / "config.json").write_text("{}", encoding="utf-8")
        (model_dir / "tokenizer.json").write_text("{}", encoding="utf-8")
        (model_dir / "pytorch_model.bin").write_bytes(b"weights")
        (model_dir / "onnx" / "model.onnx").write_bytes(b"onnx")
        (model_dir / "onnx" / "model.onnx_data").write_bytes(b"onnx-data")
        return model_dir

    def backup_args(self, model_dir: Path, archive: Path) -> argparse.Namespace:
        return argparse.Namespace(model_dir=str(model_dir), archive=str(archive), replace=False)

    def restore_args(self, model_dir: Path, archive: Path) -> argparse.Namespace:
        return argparse.Namespace(model_dir=str(model_dir), archive=str(archive))

    def test_backup_and_restore_verifies_tar_gz_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = self.create_model(root / "source")
            archive = root / "archive" / "bge-m3.tar.gz"

            result = preparation.backup_model_action(self.backup_args(source, archive))
            self.assertTrue(result.ok)
            self.assertTrue(archive.is_file())

            target = root / "restored" / "bge-m3"
            restored = preparation.restore_model_action(self.restore_args(target, archive))
            self.assertTrue(restored.ok)
            self.assertEqual(preparation.validate_model(target), (True, "verified 5 files"))

    def test_backup_parser_requires_explicit_archive(self) -> None:
        with redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            preparation.parse_args(["backup-model", "--model-dir", "model-cache"])

    def test_restore_rejects_corrupt_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = self.create_model(root / "source")
            archive = root / "corrupt.tar.gz"
            manifest = preparation.build_manifest(source)
            manifest["files"][0]["sha256"] = "0" * 64
            with tarfile.open(archive, "w:gz") as output:
                for file_path in preparation.iter_model_files(source):
                    output.add(file_path, arcname=f"bge-m3/{file_path.relative_to(source).as_posix()}")
                encoded = json.dumps(manifest).encode("utf-8")
                entry = tarfile.TarInfo(f"bge-m3/{preparation.MANIFEST_NAME}")
                entry.size = len(encoded)
                output.addfile(entry, io.BytesIO(encoded))

            with self.assertRaisesRegex(preparation.PreparationError, "manifest hash mismatch"):
                preparation.restore_model_action(self.restore_args(root / "target" / "bge-m3", archive))

    def test_restore_refuses_nonempty_target(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = self.create_model(root / "source")
            archive = root / "bge-m3.tar.gz"
            preparation.backup_model_action(self.backup_args(source, archive))
            target = root / "target" / "bge-m3"
            target.mkdir(parents=True)
            (target / "existing.txt").write_text("do not replace", encoding="utf-8")

            with self.assertRaisesRegex(preparation.PreparationError, "restore target is non-empty"):
                preparation.restore_model_action(self.restore_args(target, archive))

    def test_stage_copies_verified_source_without_mutating_it(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = self.create_model(root / "source")
            target = root / "target" / "bge-m3"
            args = argparse.Namespace(source_dir=str(source), model_dir=str(target))

            result = preparation.stage_model_action(args)

            self.assertTrue(result.ok)
            self.assertTrue((source / "pytorch_model.bin").is_file())
            self.assertEqual(preparation.validate_model(target), (True, "verified 5 files"))


if __name__ == "__main__":
    unittest.main()
