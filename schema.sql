-- FitMind 第一版 PostgreSQL 表结构。
-- 用途：创建第一版所需的用户、衣物、AI 识别、穿搭推荐、反馈和事件记录表。
-- 安全说明：本文件只创建缺失的扩展、数据表、索引和更新时间触发器。
-- 本文件不会删除表，也不会清空已有数据。

BEGIN;

-- 必需扩展：
-- - pgcrypto：用于 gen_random_uuid() 生成 UUID 主键。
-- - citext：用于大小写不敏感的邮箱字段。
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- 通用更新时间触发器函数。
-- 输入：正在被更新的数据行。
-- 输出：将 updated_at 刷新为当前时间后的同一行数据。
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 应用用户表。
-- 用户是衣橱、穿搭、反馈和事件数据的根归属对象。
CREATE TABLE IF NOT EXISTS public.app_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email CITEXT,
  phone TEXT,
  password_hash TEXT,
  nickname TEXT NOT NULL DEFAULT '',
  avatar_url TEXT,
  gender TEXT NOT NULL DEFAULT 'unknown',
  height_cm SMALLINT,
  status TEXT NOT NULL DEFAULT 'active',
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,

  CONSTRAINT app_users_gender_check
    CHECK (gender IN ('female', 'male', 'non_binary', 'prefer_not_to_say', 'unknown')),
  CONSTRAINT app_users_height_check
    CHECK (height_cm IS NULL OR height_cm BETWEEN 80 AND 250),
  CONSTRAINT app_users_status_check
    CHECK (status IN ('active', 'disabled', 'deleted'))
);

CREATE UNIQUE INDEX IF NOT EXISTS app_users_email_unique
  ON public.app_users (email)
  WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS app_users_phone_unique
  ON public.app_users (phone)
  WHERE phone IS NOT NULL AND deleted_at IS NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'app_users_set_updated_at'
  ) THEN
    CREATE TRIGGER app_users_set_updated_at
    BEFORE UPDATE ON public.app_users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();
  END IF;
END;
$$;

-- 用户偏好画像表。
-- 用于存储系统学习到的偏好，以及用户主动编辑过的偏好信号。
CREATE TABLE IF NOT EXISTS public.user_preference_profiles (
  user_id UUID PRIMARY KEY REFERENCES public.app_users(id) ON DELETE CASCADE,
  preferred_colors TEXT[] NOT NULL DEFAULT '{}',
  disliked_colors TEXT[] NOT NULL DEFAULT '{}',
  preferred_styles TEXT[] NOT NULL DEFAULT '{}',
  disliked_styles TEXT[] NOT NULL DEFAULT '{}',
  preferred_categories TEXT[] NOT NULL DEFAULT '{}',
  disliked_categories TEXT[] NOT NULL DEFAULT '{}',
  preferred_scenes TEXT[] NOT NULL DEFAULT '{}',
  preferred_formality_min SMALLINT NOT NULL DEFAULT 1,
  preferred_formality_max SMALLINT NOT NULL DEFAULT 5,
  color_weights JSONB NOT NULL DEFAULT '{}',
  style_weights JSONB NOT NULL DEFAULT '{}',
  category_weights JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT user_preference_formality_min_check
    CHECK (preferred_formality_min BETWEEN 1 AND 5),
  CONSTRAINT user_preference_formality_max_check
    CHECK (preferred_formality_max BETWEEN 1 AND 5),
  CONSTRAINT user_preference_formality_range_check
    CHECK (preferred_formality_min <= preferred_formality_max)
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'user_preference_profiles_set_updated_at'
  ) THEN
    CREATE TRIGGER user_preference_profiles_set_updated_at
    BEFORE UPDATE ON public.user_preference_profiles
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();
  END IF;
END;
$$;

-- 衣物表。
-- 用于存储用户确认后的衣物状态，而不是仅保存 AI 原始识别结果。
CREATE TABLE IF NOT EXISTS public.clothing_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  image_url TEXT NOT NULL,
  thumbnail_url TEXT,
  category TEXT NOT NULL,
  sub_category TEXT,
  color_main TEXT,
  color_secondary TEXT,
  season_tags TEXT[] NOT NULL DEFAULT '{}',
  style_tags TEXT[] NOT NULL DEFAULT '{}',
  material TEXT,
  thickness TEXT NOT NULL DEFAULT 'unknown',
  fit_type TEXT NOT NULL DEFAULT 'unknown',
  formality_score SMALLINT,
  activity_level SMALLINT,
  status TEXT NOT NULL DEFAULT 'active',
  wear_count INTEGER NOT NULL DEFAULT 0,
  last_worn_at TIMESTAMPTZ,
  ai_confidence NUMERIC(4,3),
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,

  CONSTRAINT clothing_items_category_check
    CHECK (category IN ('top', 'bottom', 'outerwear', 'dress', 'shoes', 'bag', 'accessory')),
  CONSTRAINT clothing_items_thickness_check
    CHECK (thickness IN ('thin', 'regular', 'thick', 'unknown')),
  CONSTRAINT clothing_items_fit_type_check
    CHECK (fit_type IN ('slim', 'regular', 'loose', 'oversized', 'unknown')),
  CONSTRAINT clothing_items_formality_score_check
    CHECK (formality_score IS NULL OR formality_score BETWEEN 1 AND 5),
  CONSTRAINT clothing_items_activity_level_check
    CHECK (activity_level IS NULL OR activity_level BETWEEN 1 AND 5),
  CONSTRAINT clothing_items_status_check
    CHECK (status IN ('active', 'laundry', 'idle', 'not_recommended', 'deleted')),
  CONSTRAINT clothing_items_ai_confidence_check
    CHECK (ai_confidence IS NULL OR ai_confidence BETWEEN 0 AND 1)
);

CREATE INDEX IF NOT EXISTS clothing_items_user_idx
  ON public.clothing_items(user_id);

CREATE INDEX IF NOT EXISTS clothing_items_category_idx
  ON public.clothing_items(category);

CREATE INDEX IF NOT EXISTS clothing_items_status_idx
  ON public.clothing_items(status);

CREATE INDEX IF NOT EXISTS clothing_items_season_tags_idx
  ON public.clothing_items USING GIN(season_tags);

CREATE INDEX IF NOT EXISTS clothing_items_style_tags_idx
  ON public.clothing_items USING GIN(style_tags);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'clothing_items_set_updated_at'
  ) THEN
    CREATE TRIGGER clothing_items_set_updated_at
    BEFORE UPDATE ON public.clothing_items
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();
  END IF;
END;
$$;

-- AI 识别记录表。
-- 用于将 AI 原始识别结果与用户确认后的衣物数据分开保存。
CREATE TABLE IF NOT EXISTS public.ai_recognition_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  clothing_item_id UUID REFERENCES public.clothing_items(id) ON DELETE SET NULL,
  image_url TEXT NOT NULL,
  provider TEXT,
  model_name TEXT,
  request_prompt TEXT,
  result_json JSONB NOT NULL DEFAULT '{}',
  confidence NUMERIC(4,3),
  status TEXT NOT NULL DEFAULT 'succeeded',
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT ai_recognition_confidence_check
    CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
  CONSTRAINT ai_recognition_status_check
    CHECK (status IN ('pending', 'succeeded', 'failed', 'edited'))
);

CREATE INDEX IF NOT EXISTS ai_recognition_results_user_idx
  ON public.ai_recognition_results(user_id);

CREATE INDEX IF NOT EXISTS ai_recognition_results_clothing_item_idx
  ON public.ai_recognition_results(clothing_item_id);

CREATE INDEX IF NOT EXISTS ai_recognition_results_status_idx
  ON public.ai_recognition_results(status);

-- 穿搭请求表。
-- 一次用户请求可以生成多套穿搭结果。
CREATE TABLE IF NOT EXISTS public.outfit_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  scene TEXT NOT NULL,
  style_goal TEXT,
  required_item_id UUID REFERENCES public.clothing_items(id) ON DELETE SET NULL,
  temperature_c NUMERIC(4,1),
  weather TEXT,
  season TEXT,
  constraints JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT outfit_requests_scene_check
    CHECK (scene IN ('commute', 'casual', 'date', 'sport', 'travel', 'interview', 'party', 'other'))
);

CREATE INDEX IF NOT EXISTS outfit_requests_user_idx
  ON public.outfit_requests(user_id);

CREATE INDEX IF NOT EXISTS outfit_requests_scene_idx
  ON public.outfit_requests(scene);

-- 穿搭结果表。
-- 用于存储单套推荐穿搭以及对应的推荐解释。
CREATE TABLE IF NOT EXISTS public.outfits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  request_id UUID REFERENCES public.outfit_requests(id) ON DELETE SET NULL,
  scene TEXT NOT NULL,
  style_goal TEXT,
  score NUMERIC(5,2) NOT NULL DEFAULT 0,
  reason_codes TEXT[] NOT NULL DEFAULT '{}',
  recommend_reason TEXT,
  source TEXT NOT NULL DEFAULT 'rule',
  status TEXT NOT NULL DEFAULT 'generated',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,

  CONSTRAINT outfits_score_check
    CHECK (score BETWEEN 0 AND 100),
  CONSTRAINT outfits_source_check
    CHECK (source IN ('rule', 'ai', 'manual')),
  CONSTRAINT outfits_status_check
    CHECK (status IN ('generated', 'viewed', 'saved', 'liked', 'skipped', 'worn', 'deleted'))
);

CREATE INDEX IF NOT EXISTS outfits_user_idx
  ON public.outfits(user_id);

CREATE INDEX IF NOT EXISTS outfits_request_idx
  ON public.outfits(request_id);

CREATE INDEX IF NOT EXISTS outfits_status_idx
  ON public.outfits(status);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'outfits_set_updated_at'
  ) THEN
    CREATE TRIGGER outfits_set_updated_at
    BEFORE UPDATE ON public.outfits
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at();
  END IF;
END;
$$;

-- 穿搭单品关联表。
-- 用于记录每套穿搭具体使用了哪些衣物。
CREATE TABLE IF NOT EXISTS public.outfit_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  outfit_id UUID NOT NULL REFERENCES public.outfits(id) ON DELETE CASCADE,
  clothing_item_id UUID NOT NULL REFERENCES public.clothing_items(id),
  role TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,

  CONSTRAINT outfit_items_role_check
    CHECK (role IN ('top', 'bottom', 'outerwear', 'dress', 'shoes', 'bag', 'accessory')),
  CONSTRAINT outfit_items_unique_item_per_outfit
    UNIQUE (outfit_id, clothing_item_id)
);

CREATE INDEX IF NOT EXISTS outfit_items_outfit_idx
  ON public.outfit_items(outfit_id);

CREATE INDEX IF NOT EXISTS outfit_items_clothing_item_idx
  ON public.outfit_items(clothing_item_id);

-- 穿搭反馈表。
-- 用于记录用户对穿搭的显式反馈，并为后续偏好学习提供数据。
CREATE TABLE IF NOT EXISTS public.outfit_feedbacks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  outfit_id UUID NOT NULL REFERENCES public.outfits(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  from_clothing_item_id UUID REFERENCES public.clothing_items(id) ON DELETE SET NULL,
  to_clothing_item_id UUID REFERENCES public.clothing_items(id) ON DELETE SET NULL,
  reason TEXT,
  context_json JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT outfit_feedbacks_action_check
    CHECK (action IN ('view', 'like', 'skip', 'save', 'replace_item', 'wear', 'dislike', 'remove_save'))
);

CREATE INDEX IF NOT EXISTS outfit_feedbacks_user_idx
  ON public.outfit_feedbacks(user_id);

CREATE INDEX IF NOT EXISTS outfit_feedbacks_outfit_idx
  ON public.outfit_feedbacks(outfit_id);

CREATE INDEX IF NOT EXISTS outfit_feedbacks_action_idx
  ON public.outfit_feedbacks(action);

-- 通用用户事件表。
-- 用于记录产品行为分析和推荐学习所需的用户行为信号。
CREATE TABLE IF NOT EXISTS public.user_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES public.app_users(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID,
  context_json JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS user_events_user_idx
  ON public.user_events(user_id);

CREATE INDEX IF NOT EXISTS user_events_event_type_idx
  ON public.user_events(event_type);

CREATE INDEX IF NOT EXISTS user_events_target_idx
  ON public.user_events(target_type, target_id);

COMMIT;
