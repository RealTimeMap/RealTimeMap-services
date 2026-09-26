-- Наполнение каталога достижений: метки, комментарии, баги и уникальные награды.
--
-- Зачем файлом, а не в seed.go: seed.go заводит минимум, без которого сервис
-- не работает (правило опыта за комментарий и его цепочка). Каталог наград —
-- продуктовый контент: он меняется маркетингом, а не разработкой, и держать
-- его в коде значит выкатывать сервис ради правки текста достижения.
--
-- Скрипт идемпотентен: повторный запуск не создаёт дублей и не перетирает
-- уже открытые пользователям достижения. Тексты и награды существующих
-- записей обновляются — это позволяет править формулировки, прогоняя файл
-- заново. Пороги (threshold) и триггеры (trigger_event_type) НЕ обновляются:
-- их изменение задним числом либо отняло бы у пользователя полученную
-- награду, либо выдало бы её тем, кто условие не выполнял.
--
-- Запуск:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f 0002_seed_achievements.sql
--
-- Порядок: запускать после 0001 и после первого старта сервиса — таблицы
-- создаёт AutoMigrate.
--
-- Разделение с seed.go: там остались только правило опыта за комментарий и
-- его цепочка (first_comment … commenter_100) — без них сервис не начисляет
-- ничего. Всё остальное живёт здесь. Заводить одно достижение в обоих местах
-- нельзя: коды достижений совпали бы, а коды наград разошлись, и в базе
-- остались бы неиспользуемые xp_rewards.

BEGIN;

-- ---------------------------------------------------------------------------
-- Награды опыта
-- ---------------------------------------------------------------------------
--
-- Размер награды растёт с порогом нелинейно: 10 меток — рутина, 500 — год
-- активности. Линейная шкала обесценила бы верхние ступени.
--
-- counter_only (0 XP) — служебная награда для правил, заведённых ради
-- счётчика достижений. Счётчик инкрементится внутри GreatUserExp уже после
-- проверки правила, поэтому без правила порог недостижим, а начислять опыт
-- второй раз за то же действие незачем.

INSERT INTO xp_rewards (code, amount, description, created_at, updated_at)
VALUES
    ('counter_only',            0,   'Нулевая награда для правил-счётчиков',  now(), now()),

    -- Метки
    ('ach_first_mark',          15,  'Награда за первую метку',               now(), now()),
    ('ach_mark_10',             30,  'Награда за 10 меток',                   now(), now()),
    ('ach_mark_50',             75,  'Награда за 50 меток',                   now(), now()),
    ('ach_mark_100',            150, 'Награда за 100 меток',                  now(), now()),
    ('ach_mark_500',            400, 'Награда за 500 меток',                  now(), now()),

    -- Комментарии (ступени 250 и 500 дополняют цепочку из seed.go)
    ('ach_commenter_250',       200, 'Награда за 250 комментариев',           now(), now()),
    ('ach_commenter_500',       350, 'Награда за 500 комментариев',           now(), now()),

    -- Баги. bug_confirmed — опыт за каждый подтверждённый разработчиком
    -- отчёт: засчитывается только воспроизведённый баг, а большинство
    -- отчётов проверку не проходит, поэтому награда выше, чем за комментарий.
    ('bug_confirmed',           30,  'Опыт за подтверждённый баг',            now(), now()),
    ('ach_first_bug',           25,  'Награда за первый подтверждённый баг',  now(), now()),
    ('ach_bug_5',               60,  'Награда за 5 подтверждённых багов',     now(), now()),
    ('ach_bug_25',              150, 'Награда за 25 подтверждённых багов',    now(), now()),
    ('ach_bug_100',             500, 'Награда за 100 подтверждённых багов',   now(), now()),

    -- Уникальные
    ('ach_beta_tester',         250, 'Награда участнику бета-теста',          now(), now()),
    ('ach_night_owl',           20,  'Награда за метку, созданную ночью',     now(), now()),
    ('ach_early_riser',         15,  'Награда за метку, созданную ранним утром', now(), now()),
    ('ach_social_starter',      15,  'Награда за первую подписку',            now(), now()),
    ('ach_follower_10',         50,  'Награда за 10 подписок',                now(), now()),
    ('ach_follower_50',         150, 'Награда за 50 подписок',                now(), now())
ON CONFLICT (code) DO UPDATE
SET amount      = EXCLUDED.amount,
    description = EXCLUDED.description,
    updated_at  = now();

-- ---------------------------------------------------------------------------
-- Правила начисления
-- ---------------------------------------------------------------------------
--
-- Правило обязательно для КАЖДОГО типа события, по которому считается
-- достижение: user_achievement_counts инкрементится в GreatUserExp после
-- проверки правила, и без него счётчик стоит на нуле навсегда.
--
-- Награда за markCreated намеренно 0: опыт за метку начисляется по правилам
-- продукта отдельно, а здесь правило нужно только чтобы рос счётчик. Если
-- опыт за метку понадобится — меняется reward_id этой строки, достижения не
-- затрагиваются.
--
-- daily_limit — предохранитель от накрутки опыта. На счётчик достижений он
-- не влияет: тот считает события, а не начисления.

INSERT INTO event_rules (event_type, kafka_event_type, description, reward_id, is_active, is_repeatable, daily_limit, created_at, updated_at)
SELECT v.event_type, v.event_type, v.description, r.id, true, true, v.daily_limit, now(), now()
FROM (VALUES
    ('markCreated',               'Счётчик созданных меток',              'counter_only', 30::bigint),
    ('markCreatedAtNight',        'Счётчик ночных меток',                 'counter_only', NULL::bigint),
    ('markCreatedAtEarlyMorning', 'Счётчик утренних меток',               'counter_only', NULL::bigint),
    -- Без дневного лимита: событие вызывает разработчик, подтвердив отчёт, и
    -- автор накрутить его не может. Повторы отсекает feedback-service —
    -- bug.confirmed уходит только при первом подтверждении бага.
    ('bug.confirmed',             'Опыт за баг, подтверждённый разработчиком', 'bug_confirmed', NULL::bigint),
    ('subscription.created',      'Счётчик новых подписок',               'counter_only', NULL::bigint)
) AS v(event_type, description, reward_code, daily_limit)
JOIN xp_rewards r ON r.code = v.reward_code
ON CONFLICT (event_type) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Достижения
-- ---------------------------------------------------------------------------
--
-- Прогрессивные цепочки (метки, комментарии, баги) связываются через next_id
-- ниже: выдача ближайших достижений показывает только текущую ступень, а не
-- все сразу.
--
-- Уникальные достижения в цепочки не входят — они независимы и порядка не
-- имеют.
--
-- Иконки — имена из набора проекта (preview.html), клиент рисует их сам.

INSERT INTO achievements (code, title, "desc", trigger_event_type, threshold, icon, is_active, reward_id, created_at, updated_at)
SELECT v.code, v.title, v.descr, v.trigger, v.threshold, v.icon, v.is_active, r.id, now(), now()
FROM (VALUES
    -- Метки: от первой до пятисот.
    ('first_mark',      'Первый шаг',        'Создайте свою первую метку на карте',        'markCreated', 1::bigint,   'app:flag-loop',          true,  'ach_first_mark'),
    ('mark_10',         'Картограф',         'Создайте 10 меток',                          'markCreated', 10::bigint,  'app:map-pin-star-loop',  true,  'ach_mark_10'),
    ('mark_50',         'Исследователь',     'Создайте 50 меток',                          'markCreated', 50::bigint,  'app:compass-loop',       true,  'ach_mark_50'),
    ('mark_100',        'Путешественник',    'Создайте 100 меток',                         'markCreated', 100::bigint, 'app:route-loop',         true,  'ach_mark_100'),
    ('mark_500',        'Весь мир',          'Создайте 500 меток',                         'markCreated', 500::bigint, 'app:globe-loop',         true,  'ach_mark_500'),

    -- Комментарии: продолжение цепочки из seed.go (1, 10, 50, 100).
    ('commenter_250',   'Летописец',         'Оставьте 250 комментариев',                  'comment.created', 250::bigint, 'app:certificate-loop', true, 'ach_commenter_250'),
    ('commenter_500',   'Голос города',      'Оставьте 500 комментариев',                  'comment.created', 500::bigint, 'app:city',             true, 'ach_commenter_500'),

    -- Баги: считаются только отчёты, которые разработчик подтвердил, —
    -- большинство отчётов не воспроизводится, и награда за сам факт отчёта
    -- поощряла бы пустые.
    ('first_bug',       'Первый баг',        'Найдите ошибку, которую подтвердит разработчик', 'bug.confirmed', 1::bigint,   'app:bug-hunter-loop', true, 'ach_first_bug'),
    ('bug_5',           'Внимательный глаз', 'Найдите 5 подтверждённых ошибок',                'bug.confirmed', 5::bigint,   'app:target-loop',     true, 'ach_bug_5'),
    ('bug_25',          'Охотник за багами', 'Найдите 25 подтверждённых ошибок',               'bug.confirmed', 25::bigint,  'app:gem-loop',        true, 'ach_bug_25'),
    ('bug_100',         'Гроза ошибок',      'Найдите 100 подтверждённых ошибок',              'bug.confirmed', 100::bigint, 'app:summit-loop',     true, 'ach_bug_100'),

    -- Подписки — ИСХОДЯЩИЕ, а не подписчики.
    --
    -- В событии subscription.created gamification-service видит пользователя
    -- только из заголовка, а туда social-service кладёт SubscriberID — того,
    -- кто подписался (TargetID уходит в SourceID и в payload, который здесь
    -- не разбирается). Достижения «получите N подписчиков» на этом событии
    -- доставались бы не тем: их условием стало бы «подпишитесь на N».
    --
    -- Награды за подписчиков нужны — но им нужно своё событие с получателем
    -- в заголовке. См. примечание 3 в конце файла.
    ('social_starter',  'Первое знакомство', 'Подпишитесь на первого пользователя',        'subscription.created', 1::bigint,  'app:people-loop', true, 'ach_social_starter'),
    ('follower_10',     'Любознательный',    'Подпишитесь на 10 пользователей',            'subscription.created', 10::bigint, 'app:heart-loop',  true, 'ach_follower_10'),
    ('follower_50',     'Сетевой житель',    'Подпишитесь на 50 пользователей',            'subscription.created', 50::bigint, 'app:share-loop',  true, 'ach_follower_50'),

    -- Уникальные: вне цепочек, порог 1.
    ('beta_tester',     'Бета-тестер',       'Вы были с нами на этапе бета-теста. Спасибо!', 'beta.participant', 1::bigint, 'app:beta-flask-loop', true, 'ach_beta_tester'),
    ('night_mark',      'Полуночник',        'Создайте метку между 00:00 и 05:00',         'markCreatedAtNight',        1::bigint, 'app:night-loop',       true, 'ach_night_owl'),
    ('early_bird_mark', 'Ранняя пташка',     'Создайте метку между 05:00 и 08:00',         'markCreatedAtEarlyMorning', 1::bigint, 'app:early-bird-loop',  true, 'ach_early_riser')
) AS v(code, title, descr, trigger, threshold, icon, is_active, reward_code)
JOIN xp_rewards r ON r.code = v.reward_code
ON CONFLICT (code) DO UPDATE
SET title      = EXCLUDED.title,
    "desc"     = EXCLUDED."desc",
    icon       = EXCLUDED.icon,
    reward_id  = EXCLUDED.reward_id,
    updated_at = now();
    -- threshold, trigger_event_type и is_active намеренно не обновляются:
    -- смена порога задним числом отняла бы награду у тех, кто её получил,
    -- либо выдала бы её невыполнившим. Менять их — отдельной миграцией,
    -- осознанно.

-- ---------------------------------------------------------------------------
-- Цепочки
-- ---------------------------------------------------------------------------
--
-- Связывание отдельным проходом: next_id ссылается на достижение, которого
-- при вставке предыдущего ещё не существует.
--
-- Обновляется только пустой next_id — ручная перестройка цепочки не
-- перетирается. Из-за этого первый запуск после добавления новых ступеней
-- прошьёт связь commenter_100 → commenter_250, а повторный ничего не
-- изменит.

UPDATE achievements a
SET next_id = n.id, updated_at = now()
FROM (VALUES
    -- Метки
    ('first_mark',      'mark_10'),
    ('mark_10',         'mark_50'),
    ('mark_50',         'mark_100'),
    ('mark_100',        'mark_500'),

    -- Комментарии: хвост цепочки из seed.go
    ('commenter_100',   'commenter_250'),
    ('commenter_250',   'commenter_500'),

    -- Баги
    ('first_bug',       'bug_5'),
    ('bug_5',           'bug_25'),
    ('bug_25',          'bug_100'),

    -- Подписки
    ('social_starter',  'follower_10'),
    ('follower_10',     'follower_50')
) AS chain(from_code, to_code)
JOIN achievements n ON n.code = chain.to_code AND n.deleted_at IS NULL
WHERE a.code = chain.from_code
  AND a.deleted_at IS NULL
  AND a.next_id IS NULL;

COMMIT;

-- ---------------------------------------------------------------------------
-- Что требует внимания после применения
-- ---------------------------------------------------------------------------
--
-- 1. beta_tester не выдаётся автоматически: события beta.participant не
--    существует. Достижение выдаётся вручную — вставкой в user_achievements
--    списку участников беты, например:
--
--      INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
--      SELECT u.id, a.id, now()
--      FROM (VALUES (1),(2),(3)) AS u(id)
--      CROSS JOIN achievements a
--      WHERE a.code = 'beta_tester'
--      ON CONFLICT DO NOTHING;
--
--    Опыт при таком способе не начисляется — XP-операцию нужно завести
--    отдельно либо смириться, что награда здесь чисто символическая.
--
-- 2. Достижения на баги считаются по bug.confirmed: feedback-service
--    публикует его, когда разработчик впервые подтверждает баг с автором.
--    Базу, где этот файл уже применялся с прежним bug.created, переводит
--    0003_bug_achievements_confirmed.sql.
--
-- 3. Достижений «получите N подписчиков» здесь нет намеренно, хотя они
--    напрашиваются. Причина техническая: gamification-service определяет
--    пользователя события через parseBody → заголовок, а в заголовке
--    subscription.created лежит SubscriberID (подписавшийся). Получатель
--    подписки уходит в SourceID и в payload.targetId, но payload этого
--    события там не разбирается — в parseBody перечислены userId, user_id,
--    commentId, markId, source_id.
--
--    Чтобы наградить за подписчиков, нужно одно из двух:
--      * отдельное событие (например subscriberGained) с получателем в
--        заголовке UserID — по образцу markCreatedAtNight;
--      * разбор payload.targetId в parseBody с привязкой к типу события.
--
--    Первое проще и не трогает общий разбор. До этого момента награды за
--    подписчиков выдать нечем.
