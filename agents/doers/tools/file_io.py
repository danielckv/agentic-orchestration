import os

class WorktreeFileIO:
    """File I/O scoped to a git worktree path."""

    def __init__(self, worktree_path: str):
        self.root = os.path.abspath(worktree_path)

    def _safe_path(self, path: str) -> str:
        full = os.path.abspath(os.path.join(self.root, path))
        if not full.startswith(self.root):
            raise ValueError(f"Path escapes worktree: {path}")
        return full

    def read(self, path: str) -> str:
        return open(self._safe_path(path)).read()

    def write(self, path: str, content: str) -> None:
        full = self._safe_path(path)
        os.makedirs(os.path.dirname(full), exist_ok=True)
        with open(full, "w") as f:
            f.write(content)

    def exists(self, path: str) -> bool:
        return os.path.exists(self._safe_path(path))

    def list_dir(self, path: str = ".") -> list[str]:
        return os.listdir(self._safe_path(path))
