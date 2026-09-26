-- Перевод достижений за баги на bug.confirmed и уборка дублей.
--
-- Зачем: цепочка багов (first_bug … bug_100) заводилась выключенной и
-- считалась по bug.created, которое никто не публикует. feedback-service
-- публикует bug.confirmed — когда разработчик впервые подтверждает баг с
-- автором. Кроме того, одна из версий seed.go завела собственную цепочку
-- багов (bug_hunter_5/15/30) с тем же кодом first_bug — её нужно убрать,
-- каталог достижений живёт в 0002.
--
-- Скрипт рассчитан на любое из состояний базы:
--   * применена прежняя 0002 (bug.created, is_active = false);
--   * отработал seed.go с цепочкой bug_hunter_*;
--   * и то и другое, либо уже новая 0002 — тогда скрипт ничего не меняет.
-- Повторный запуск безопасен.
--
-- Смена trigger_event_type и threshold задним числом здесь безопасна, хотя
-- 0002 такого избегает: по bug.created не пришло ни одного события, так что
-- открыть эти достижения никто не мог.
--
-- Запуск (после 0002):
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f 0003_bug_achievements_confirmed.sql

BEGIN;

-- Награды: опыт за подтверждённый баг и награды ступеней цепочки.
INSERT INTO xp_rewards (code, amount, description, created_at, updated_at)
VALUES
    ('bug_confirmed', 30,  'Опыт за подтверждённый баг',           now(), now()),
    ('ach_first_bug', 25,  'Награда за первый подтверждённый баг', now(), now()),
    ('ach_bug_5',     60,  'Награда за 5 подтверждённых багов',    now(), now()),
    ('ach_bug_25',    150, 'Награда за 25 подтверждённых багов',   now(), now()),
    ('ach_bug_100',   500, 'Награда за 100 подтверждённых багов',  now(), now())
ON CONFLICT (code) DO UPDATE
SET amount      = EXCLUDED.amount,
    description = EXCLUDED.description,
    updated_at  = now();

-- Правило bug.confirmed: без него счётчик достижений не растёт.
INSERT INTO event_rules (event_type, kafka_event_type, description, reward_id, is_active, is_repeatable, daily_limit, created_at, updated_at)
SELECT 'bug.confirmed', 'bug.confirmed', 'Опыт за баг, подтверждённый разработчиком', r.id, true, true, NULL, now(), now()
FROM xp_rewards r
WHERE r.code = 'bug_confirmed'
ON CONFLICT (event_type) DO NOTHING;

-- bug.created больше не нужен: такого события нет и не будет.
UPDATE event_rules
SET is_active = false, updated_at = now()
WHERE event_type = 'bug.created' AND is_active;

-- Цепочка багов: заводим недостающие ступени и приводим существующие к
-- bug.confirmed. Здесь, в отличие от 0002, триггер, порог и is_active
-- обновляются — см. шапку.
INSERT INTO achievements (code, title, "desc", trigger_event_type, threshold, icon, is_active, reward_id, created_at, updated_at)
SELECT v.code, v.title, v.descr, 'bug.confirmed', v.threshold, v.icon, true, r.id, now(), now()
FROM (VALUES
    ('first_bug', 'Первый баг',        'Найдите ошибку, которую подтвердит разработчик', 1::bigint,   'app:bug-hunter-loop', 'ach_first_bug'),
    ('bug_5',     'Внимательный глаз', 'Найдите 5 подтверждённых ошибок',                5::bigint,   'app:target-loop',     'ach_bug_5'),
    ('bug_25',    'Охотник за багами', 'Найдите 25 подтверждённых ошибок',               25::bigint,  'app:gem-loop',        'ach_bug_25'),
    ('bug_100',   'Гроза ошибок',      'Найдите 100 подтверждённых ошибок',              100::bigint, 'app:summit-loop',     'ach_bug_100')
) AS v(code, title, descr, threshold, icon, reward_code)
JOIN xp_rewards r ON r.code = v.reward_code
ON CONFLICT (code) DO UPDATE
SET title              = EXCLUDED.title,
    "desc"             = EXCLUDED."desc",
    trigger_event_type = EXCLUDED.trigger_event_type,
    threshold          = EXCLUDED.threshold,
    icon               = EXCLUDED.icon,
    is_active          = true,
    reward_id          = EXCLUDED.reward_id,
    deleted_at         = NULL,
    updated_at         = now();

-- Дубли из seed.go убираем мягким удалением. Открытые кому-то ступени не
-- трогаем, чтобы не отнять полученное, — о них скажет NOTICE ниже.
UPDATE achievements a
SET deleted_at = now(), next_id = NULL, updated_at = now()
WHERE a.code IN ('bug_hunter_5', 'bug_hunter_15', 'bug_hunter_30')
  AND a.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM user_achievements ua WHERE ua.achievement_id = a.id);

-- Связи цепочки ставятся принудительно, а не только в пустой next_id:
-- seed.go мог связать first_bug с bug_hunter_5.
UPDATE achievements a
SET next_id = n.id, updated_at = now()
FROM (VALUES
    ('first_bug', 'bug_5'),
    ('bug_5',     'bug_25'),
    ('bug_25',    'bug_100')
) AS chain(from_code, to_code)
JOIN achievements n ON n.code = chain.to_code AND n.deleted_at IS NULL
WHERE a.code = chain.from_code
  AND a.deleted_at IS NULL
  AND a.next_id IS DISTINCT FROM n.id;

DO $$
DECLARE
    kept int;
BEGIN
    SELECT count(*) INTO kept
    FROM achievements
    WHERE code IN ('bug_hunter_5', 'bug_hunter_15', 'bug_hunter_30')
      AND deleted_at IS NULL;

    IF kept > 0 THEN
        RAISE NOTICE 'Оставлено достижений bug_hunter_*, уже открытых пользователям: %. Решите вручную.', kept;
    END IF;
END $$;

COMMIT;
