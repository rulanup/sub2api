UPDATE groups
SET is_exclusive = false,
    updated_at = NOW()
WHERE is_private = true
  AND is_exclusive = true;
