UPDATE groups
SET rate_multiplier = 0,
    image_rate_independent = false,
    image_rate_multiplier = 0,
    video_rate_independent = false,
    video_rate_multiplier = 0,
    updated_at = NOW()
WHERE is_private = true;
