-- Перевод achievements.icon с загруженного файла на имя иконки iconfy.
--
-- Зачем: иконку достижения рисует клиент по имени ("mdi:trophy-outline"),
-- поэтому хранить и раздавать файл серверу больше незачем.
--
-- Колонка icon хранила bytea с JSON-сериализацией types.Photo внутри.
-- Старые значения не конвертируются, а обнуляются: в JSON лежит URL файла,
-- а не имя иконки — превратить одно в другое нельзя. Иконки достижений
-- нужно проставить заново через API.
--
-- Скрипт идемпотентен и рассчитан на оба состояния схемы:
--   * icon ещё bytea — меняем тип и сбрасываем значения;
--   * icon уже varchar — значит AutoMigrate успел отработать раньше этой
--     миграции. Он меняет тип сам, приводя bytea к тексту, и в колонке
--     остаётся hex-дамп прежнего JSON, обрезанный до 128 символов. Такие
--     значения тоже чистим.
--
-- Запуск:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f 0001_achievement_icon_string.sql

BEGIN;

DO $$
DECLARE
    icon_type text;
    cleaned   bigint;
BEGIN
    SELECT data_type INTO icon_type
    FROM information_schema.columns
    WHERE table_name = 'achievements' AND column_name = 'icon';

    IF icon_type IS NULL THEN
        RAISE NOTICE 'Таблицы achievements нет — миграция не нужна, схему создаст AutoMigrate.';
        RETURN;
    END IF;

    IF icon_type <> 'character varying' THEN
        -- USING NULL, а не приведение типа: в колонке лежит JSON с URL файла,
        -- осмысленного имени иконки из него не получить.
        ALTER TABLE achievements
            ALTER COLUMN icon TYPE varchar(128) USING NULL;

        RAISE NOTICE 'achievements.icon переведён в varchar(128), значения сброшены.';
        RETURN;
    END IF;

    -- Колонка уже varchar: чистим то, что осталось от прежнего формата.
    -- Имя иконки iconfy не начинается ни с "\x" (hex-дамп bytea), ни с "{"
    -- (сериализованный JSON), поэтому реальные значения под условие не попадут.
    UPDATE achievements
    SET icon = NULL
    WHERE icon IS NOT NULL
      AND (icon LIKE E'\\\\x%' OR icon LIKE '{%');

    GET DIAGNOSTICS cleaned = ROW_COUNT;

    RAISE NOTICE 'achievements.icon уже varchar; очищено остатков прежнего формата: %.', cleaned;
END $$;

-- icon_url — колонка от прежней схемы хранения. Модель её не описывает,
-- код не читает; удаляется здесь, чтобы не тянуть мёртвое поле дальше.
ALTER TABLE achievements DROP COLUMN IF EXISTS icon_url;

COMMIT;
