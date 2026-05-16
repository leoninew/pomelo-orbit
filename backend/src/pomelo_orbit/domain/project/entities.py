from dataclasses import dataclass
from datetime import datetime


@dataclass
class Project:
    id: str
    name: str
    code: str
    owner_user_id: str
    is_active: bool
    created_at: datetime
    updated_at: datetime
