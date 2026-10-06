-- Extracted from Jan's real VOA v2.7.0 fixture; schema snapshot for tests only.
CREATE TABLE `api_tokens` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `token_salt` TEXT NULL, `token_hash` TEXT NULL, `token_last_eight` TEXT NULL, `token_sha256` TEXT NULL, `permissions` TEXT NOT NULL, `expires_at` DATETIME NOT NULL, `created` DATETIME NOT NULL, `owner_id` INTEGER NOT NULL);

CREATE TABLE `buckets` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `project_view_id` INTEGER NOT NULL, `limit` INTEGER DEFAULT 0 NULL, `position` REAL NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL, `created_by_id` INTEGER NOT NULL);

CREATE TABLE `favorites` (`entity_id` INTEGER NOT NULL, `user_id` INTEGER NOT NULL, `kind` INTEGER NOT NULL, PRIMARY KEY (`entity_id`,`user_id`,`kind`));

CREATE TABLE `files` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `name` TEXT NOT NULL, `mime` TEXT NULL, `size` INTEGER NOT NULL, `created` DATETIME NULL, `created_by_id` INTEGER NOT NULL);

CREATE TABLE `label_tasks` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `task_id` INTEGER NOT NULL, `label_id` INTEGER NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `labels` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `description` TEXT NULL, `hex_color` TEXT NULL, `created_by_id` INTEGER NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `license_status` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `instance_id` TEXT NOT NULL, `response` TEXT NOT NULL, `validated_at` DATETIME NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `link_shares` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `hash` TEXT NOT NULL, `name` TEXT NULL, `project_id` INTEGER NOT NULL, `permission` INTEGER DEFAULT 0 NOT NULL, `sharing_type` INTEGER DEFAULT 0 NOT NULL, `password` TEXT NULL, `shared_by_id` INTEGER NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `migration` (`id` TEXT NULL, `description` TEXT NULL);

CREATE TABLE `migration_status` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `migrator_name` TEXT NULL, `started_at` DATETIME NOT NULL, `finished_at` DATETIME NULL, `error_kind` TEXT NULL, `error_message` TEXT NULL, `heartbeat_at` DATETIME NULL, `upload_file_id` INTEGER NULL, `active_user_id` INTEGER NULL);

CREATE TABLE `notifications` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `notifiable_id` INTEGER NOT NULL, `notification` TEXT NOT NULL, `name` TEXT NOT NULL, `subject_id` INTEGER NULL, `project_id` INTEGER DEFAULT 0 NOT NULL, `read_at` DATETIME NULL, `created` DATETIME NOT NULL);

CREATE TABLE `oauth_codes` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `code` TEXT NOT NULL, `expires_at` DATETIME NOT NULL, `client_id` TEXT NOT NULL, `redirect_uri` TEXT NOT NULL, `code_challenge` TEXT NOT NULL, `code_challenge_method` TEXT NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `project_ancestors` (`ancestor_id` INTEGER NOT NULL, `project_id` INTEGER NOT NULL, `depth` INTEGER NOT NULL, PRIMARY KEY (`ancestor_id`,`project_id`));

CREATE TABLE `project_task_counters` (`project_id` INTEGER PRIMARY KEY NOT NULL, `last_index` INTEGER DEFAULT 0 NOT NULL);

CREATE TABLE `project_views` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `project_id` INTEGER NOT NULL, `view_kind` INTEGER NOT NULL, `filter` TEXT DEFAULT null NULL, `position` REAL NULL, `bucket_configuration_mode` INTEGER DEFAULT 0 NULL, `bucket_configuration` TEXT NULL, `default_bucket_id` INTEGER NULL, `done_bucket_id` INTEGER NULL, `updated` DATETIME NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `projects` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `description` TEXT NULL, `identifier` TEXT NULL, `hex_color` TEXT NULL, `owner_id` INTEGER NOT NULL, `parent_project_id` INTEGER NULL, `is_archived` INTEGER DEFAULT 0 NOT NULL, `background_file_id` INTEGER NULL, `background_blur_hash` TEXT NULL, `position` REAL NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `reactions` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `entity_id` INTEGER NOT NULL, `entity_kind` INTEGER NOT NULL, `value` TEXT NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `saved_filters` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `filters` TEXT NOT NULL, `title` TEXT NOT NULL, `description` TEXT NULL, `owner_id` INTEGER NOT NULL, `is_favorite` INTEGER DEFAULT 0 NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `sessions` (`id` TEXT PRIMARY KEY NOT NULL, `user_id` INTEGER NOT NULL, `token_hash` TEXT NOT NULL, `device_info` TEXT NULL, `ip_address` TEXT NULL, `is_long_session` INTEGER DEFAULT 0 NOT NULL, `oidcid_token` TEXT NULL, `oidc_provider_key` TEXT NULL, `last_active` DATETIME NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `subscriptions` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `entity_type` INTEGER NOT NULL, `entity_id` INTEGER NOT NULL, `user_id` INTEGER NOT NULL, `muted` INTEGER DEFAULT 0 NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `task_assignees` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `task_id` INTEGER NOT NULL, `user_id` INTEGER NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `task_attachments` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `task_id` INTEGER NOT NULL, `file_id` INTEGER NOT NULL, `created_by_id` INTEGER NOT NULL, `created` DATETIME NULL);

CREATE TABLE `task_buckets` (`bucket_id` INTEGER NOT NULL, `task_id` INTEGER NOT NULL, `project_view_id` INTEGER NOT NULL);

CREATE TABLE `task_comments` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `comment` TEXT NOT NULL, `author_id` INTEGER NOT NULL, `task_id` INTEGER NOT NULL, `created` DATETIME NULL, `updated` DATETIME NULL);

CREATE TABLE `task_index_aliases` (`project_id` INTEGER NOT NULL, `index` INTEGER NOT NULL, `task_id` INTEGER NOT NULL);

CREATE TABLE `task_positions` (`task_id` INTEGER NOT NULL, `project_view_id` INTEGER NOT NULL, `position` REAL NOT NULL);

CREATE TABLE `task_relations` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `task_id` INTEGER NOT NULL, `other_task_id` INTEGER NOT NULL, `relation_kind` TEXT NOT NULL, `created_by_id` INTEGER NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `task_reminders` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `task_id` INTEGER NOT NULL, `reminder` DATETIME NOT NULL, `created` DATETIME NOT NULL, `relative_period` INTEGER NULL, `relative_to` TEXT NULL);

CREATE TABLE `task_unread_statuses` (`task_id` INTEGER NOT NULL, `user_id` INTEGER NOT NULL);

CREATE TABLE `tasks` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `title` TEXT NOT NULL, `description` TEXT NULL, `project_id` INTEGER NOT NULL, `done` INTEGER NULL, `done_at` DATETIME NULL, `due_date` DATETIME NULL, `repeat_after` INTEGER NULL, `repeat_mode` INTEGER DEFAULT 0 NOT NULL, `priority` INTEGER NULL, `start_date` DATETIME NULL, `end_date` DATETIME NULL, `hex_color` TEXT NULL, `percent_done` REAL NULL, `index` INTEGER DEFAULT 0 NOT NULL, `uid` TEXT NULL, `cover_image_attachment_id` INTEGER DEFAULT 0 NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL, `deleted_at` DATETIME NULL, `created_by_id` INTEGER NOT NULL);

CREATE TABLE `team_members` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `team_id` INTEGER NOT NULL, `user_id` INTEGER NOT NULL, `admin` INTEGER NULL, `created` DATETIME NOT NULL);

CREATE TABLE `team_projects` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `team_id` INTEGER NOT NULL, `project_id` INTEGER NOT NULL, `permission` INTEGER DEFAULT 0 NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `teams` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `name` TEXT NOT NULL, `description` TEXT NULL, `created_by_id` INTEGER NOT NULL, `external_id` TEXT NULL, `issuer` TEXT NULL, `created` DATETIME NULL, `updated` DATETIME NULL, `is_public` INTEGER DEFAULT 0 NOT NULL);

CREATE TABLE `time_entries` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `task_id` INTEGER NULL, `project_id` INTEGER NULL, `start_time` DATETIME NOT NULL, `end_time` DATETIME NULL, `comment` TEXT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `totp` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `secret` TEXT NOT NULL, `enabled` INTEGER NULL, `url` TEXT NULL);

CREATE TABLE `unsplash_photos` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `file_id` INTEGER NOT NULL, `unsplash_id` TEXT NULL, `author` TEXT NULL, `author_name` TEXT NULL);

CREATE TABLE `user_invite_link_teams` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `invite_link_id` INTEGER NOT NULL, `team_id` INTEGER NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `user_invite_links` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `name` TEXT NOT NULL, `token_hash` TEXT NOT NULL, `max_uses` INTEGER NULL, `uses` INTEGER DEFAULT 0 NOT NULL, `expires_at` DATETIME NULL, `skip_email_confirm` INTEGER DEFAULT 0 NOT NULL, `created_by_id` INTEGER NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `user_tokens` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `token` TEXT NOT NULL, `kind` INTEGER NOT NULL, `created` DATETIME NOT NULL);

CREATE TABLE `users` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `name` TEXT NULL, `username` TEXT NOT NULL, `password` TEXT NULL, `email` TEXT NULL, `pending_email` TEXT NULL, `status` INTEGER DEFAULT 0 NULL, `is_admin` INTEGER DEFAULT 0 NOT NULL, `avatar_provider` TEXT NULL, `avatar_file_id` INTEGER NULL, `issuer` TEXT NULL, `subject` TEXT NULL, `email_reminders_enabled` INTEGER DEFAULT 1 NULL, `discoverable_by_name` INTEGER DEFAULT 0 NULL, `discoverable_by_email` INTEGER DEFAULT 0 NULL, `overdue_tasks_reminders_enabled` INTEGER DEFAULT 1 NULL, `overdue_tasks_reminders_time` TEXT DEFAULT '09:00' NOT NULL, `default_project_id` INTEGER NULL, `bot_owner_id` INTEGER NULL, `week_start` INTEGER NULL, `language` TEXT NULL, `timezone` TEXT NULL, `deletion_scheduled_at` DATETIME NULL, `deletion_last_reminder_sent` DATETIME NULL, `frontend_settings` TEXT NULL, `extra_settings_links` TEXT NULL, `export_file_id` INTEGER NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `users_projects` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `user_id` INTEGER NOT NULL, `project_id` INTEGER NOT NULL, `permission` INTEGER DEFAULT 0 NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE TABLE `webhooks` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `target_url` TEXT NOT NULL, `events` TEXT NOT NULL, `project_id` INTEGER NULL, `user_id` INTEGER NULL, `secret` TEXT NULL, `basic_auth_user` TEXT NULL, `basic_auth_password` TEXT NULL, `created_by_id` INTEGER NOT NULL, `created` DATETIME NOT NULL, `updated` DATETIME NOT NULL);

CREATE INDEX `IDX_api_tokens_token_last_eight` ON `api_tokens` (`token_last_eight`);

CREATE INDEX `IDX_label_tasks_label_id` ON `label_tasks` (`label_id`);

CREATE INDEX `IDX_label_tasks_task_id` ON `label_tasks` (`task_id`);

CREATE INDEX `IDX_link_shares_permission` ON `link_shares` (`permission`);

CREATE INDEX `IDX_link_shares_shared_by_id` ON `link_shares` (`shared_by_id`);

CREATE INDEX `IDX_link_shares_sharing_type` ON `link_shares` (`sharing_type`);

CREATE INDEX `IDX_notifications_name` ON `notifications` (`name`);

CREATE INDEX `IDX_notifications_project_id` ON `notifications` (`project_id`);

CREATE INDEX `IDX_project_ancestors_project_id` ON `project_ancestors` (`project_id`);

CREATE INDEX `IDX_project_views_default_bucket_id` ON `project_views` (`default_bucket_id`);

CREATE INDEX `IDX_project_views_done_bucket_id` ON `project_views` (`done_bucket_id`);

CREATE INDEX `IDX_project_views_project_id` ON `project_views` (`project_id`);

CREATE INDEX `IDX_projects_owner_id` ON `projects` (`owner_id`);

CREATE INDEX `IDX_projects_parent_project_id` ON `projects` (`parent_project_id`);

CREATE INDEX `IDX_reactions_entity_id` ON `reactions` (`entity_id`);

CREATE INDEX `IDX_reactions_entity_kind` ON `reactions` (`entity_kind`);

CREATE INDEX `IDX_reactions_user_id` ON `reactions` (`user_id`);

CREATE INDEX `IDX_reactions_value` ON `reactions` (`value`);

CREATE INDEX `IDX_saved_filters_owner_id` ON `saved_filters` (`owner_id`);

CREATE INDEX `IDX_sessions_user_id` ON `sessions` (`user_id`);

CREATE INDEX `IDX_subscriptions_entity_id` ON `subscriptions` (`entity_id`);

CREATE INDEX `IDX_subscriptions_entity_type` ON `subscriptions` (`entity_type`);

CREATE INDEX `IDX_subscriptions_user_id` ON `subscriptions` (`user_id`);

CREATE INDEX `IDX_task_assignees_task_id` ON `task_assignees` (`task_id`);

CREATE INDEX `IDX_task_assignees_user_id` ON `task_assignees` (`user_id`);

CREATE INDEX `IDX_task_buckets_bucket_id` ON `task_buckets` (`bucket_id`);

CREATE INDEX `IDX_task_buckets_project_view_id` ON `task_buckets` (`project_view_id`);

CREATE INDEX `IDX_task_buckets_task_id` ON `task_buckets` (`task_id`);

CREATE INDEX `IDX_task_comments_task_id` ON `task_comments` (`task_id`);

CREATE INDEX `IDX_task_index_aliases_task_id` ON `task_index_aliases` (`task_id`);

CREATE INDEX `IDX_task_positions_project_view_id` ON `task_positions` (`project_view_id`);

CREATE INDEX `IDX_task_positions_task_id` ON `task_positions` (`task_id`);

CREATE INDEX `IDX_task_positions_view_position` ON `task_positions` (`project_view_id`,`position`);

CREATE INDEX `IDX_task_reminders_reminder` ON `task_reminders` (`reminder`);

CREATE INDEX `IDX_task_reminders_task_id` ON `task_reminders` (`task_id`);

CREATE INDEX `IDX_tasks_deleted_at` ON `tasks` (`deleted_at`);

CREATE INDEX `IDX_tasks_done` ON `tasks` (`done`);

CREATE INDEX `IDX_tasks_done_at` ON `tasks` (`done_at`);

CREATE INDEX `IDX_tasks_due_date` ON `tasks` (`due_date`);

CREATE INDEX `IDX_tasks_end_date` ON `tasks` (`end_date`);

CREATE INDEX `IDX_tasks_project_done_due_date` ON `tasks` (`project_id`,`done`,`due_date`);

CREATE INDEX `IDX_tasks_project_id` ON `tasks` (`project_id`);

CREATE INDEX `IDX_tasks_repeat_after` ON `tasks` (`repeat_after`);

CREATE INDEX `IDX_tasks_start_date` ON `tasks` (`start_date`);

CREATE INDEX `IDX_team_members_team_id` ON `team_members` (`team_id`);

CREATE INDEX `IDX_team_members_user_id` ON `team_members` (`user_id`);

CREATE INDEX `IDX_team_projects_permission` ON `team_projects` (`permission`);

CREATE INDEX `IDX_team_projects_project_id` ON `team_projects` (`project_id`);

CREATE INDEX `IDX_team_projects_team_id` ON `team_projects` (`team_id`);

CREATE INDEX `IDX_teams_created_by_id` ON `teams` (`created_by_id`);

CREATE INDEX `IDX_time_entries_project_id` ON `time_entries` (`project_id`);

CREATE INDEX `IDX_time_entries_start_time` ON `time_entries` (`start_time`);

CREATE INDEX `IDX_time_entries_task_id` ON `time_entries` (`task_id`);

CREATE INDEX `IDX_time_entries_user_id` ON `time_entries` (`user_id`);

CREATE INDEX `IDX_user_invite_link_teams_team_id` ON `user_invite_link_teams` (`team_id`);

CREATE INDEX `IDX_user_invite_links_created_by_id` ON `user_invite_links` (`created_by_id`);

CREATE INDEX `IDX_user_tokens_token` ON `user_tokens` (`token`);

CREATE INDEX `IDX_users_bot_owner_id` ON `users` (`bot_owner_id`);

CREATE INDEX `IDX_users_default_project_id` ON `users` (`default_project_id`);

CREATE INDEX `IDX_users_discoverable_by_email` ON `users` (`discoverable_by_email`);

CREATE INDEX `IDX_users_discoverable_by_name` ON `users` (`discoverable_by_name`);

CREATE INDEX `IDX_users_overdue_tasks_reminders_enabled` ON `users` (`overdue_tasks_reminders_enabled`);

CREATE INDEX `IDX_users_projects_permission` ON `users_projects` (`permission`);

CREATE INDEX `IDX_users_projects_project_id` ON `users_projects` (`project_id`);

CREATE INDEX `IDX_users_projects_user_id` ON `users_projects` (`user_id`);

CREATE INDEX `IDX_webhooks_project_id` ON `webhooks` (`project_id`);

CREATE INDEX `IDX_webhooks_user_id` ON `webhooks` (`user_id`);

CREATE UNIQUE INDEX `UQE_api_tokens_id` ON `api_tokens` (`id`);

CREATE UNIQUE INDEX `UQE_api_tokens_token_hash` ON `api_tokens` (`token_hash`);

CREATE UNIQUE INDEX `UQE_api_tokens_token_sha256` ON `api_tokens` (`token_sha256`);

CREATE UNIQUE INDEX `UQE_buckets_id` ON `buckets` (`id`);

CREATE UNIQUE INDEX `UQE_files_id` ON `files` (`id`);

CREATE UNIQUE INDEX `UQE_label_tasks_id` ON `label_tasks` (`id`);

CREATE UNIQUE INDEX `UQE_labels_id` ON `labels` (`id`);

CREATE UNIQUE INDEX `UQE_license_status_id` ON `license_status` (`id`);

CREATE UNIQUE INDEX `UQE_link_shares_hash` ON `link_shares` (`hash`);

CREATE UNIQUE INDEX `UQE_link_shares_id` ON `link_shares` (`id`);

CREATE UNIQUE INDEX `UQE_migration_status_active_user_id` ON `migration_status` (`active_user_id`);

CREATE UNIQUE INDEX `UQE_migration_status_id` ON `migration_status` (`id`);

CREATE UNIQUE INDEX `UQE_notifications_id` ON `notifications` (`id`);

CREATE UNIQUE INDEX `UQE_oauth_codes_code` ON `oauth_codes` (`code`);

CREATE UNIQUE INDEX `UQE_oauth_codes_id` ON `oauth_codes` (`id`);

CREATE UNIQUE INDEX `UQE_project_views_id` ON `project_views` (`id`);

CREATE UNIQUE INDEX `UQE_projects_id` ON `projects` (`id`);

CREATE UNIQUE INDEX `UQE_reactions_id` ON `reactions` (`id`);

CREATE UNIQUE INDEX `UQE_saved_filters_id` ON `saved_filters` (`id`);

CREATE UNIQUE INDEX `UQE_sessions_id` ON `sessions` (`id`);

CREATE UNIQUE INDEX `UQE_sessions_token_hash` ON `sessions` (`token_hash`);

CREATE UNIQUE INDEX `UQE_subscriptions_entity_user` ON `subscriptions` (`entity_type`,`entity_id`,`user_id`);

CREATE UNIQUE INDEX `UQE_subscriptions_id` ON `subscriptions` (`id`);

CREATE UNIQUE INDEX `UQE_task_assignees_id` ON `task_assignees` (`id`);

CREATE UNIQUE INDEX `UQE_task_attachments_id` ON `task_attachments` (`id`);

CREATE UNIQUE INDEX `UQE_task_buckets_task_view` ON `task_buckets` (`task_id`,`project_view_id`);

CREATE UNIQUE INDEX `UQE_task_comments_id` ON `task_comments` (`id`);

CREATE UNIQUE INDEX `UQE_task_index_aliases_task_index_alias` ON `task_index_aliases` (`project_id`,`index`);

CREATE UNIQUE INDEX `UQE_task_positions_task_view` ON `task_positions` (`task_id`,`project_view_id`);

CREATE UNIQUE INDEX `UQE_task_relations_id` ON `task_relations` (`id`);

CREATE UNIQUE INDEX `UQE_task_reminders_id` ON `task_reminders` (`id`);

CREATE UNIQUE INDEX `UQE_task_unread_statuses_task_user` ON `task_unread_statuses` (`task_id`,`user_id`);

CREATE UNIQUE INDEX `UQE_tasks_id` ON `tasks` (`id`);

CREATE UNIQUE INDEX `UQE_tasks_tasks_project_index` ON `tasks` (`project_id`,`index`);

CREATE UNIQUE INDEX `UQE_team_members_id` ON `team_members` (`id`);

CREATE UNIQUE INDEX `UQE_team_projects_id` ON `team_projects` (`id`);

CREATE UNIQUE INDEX `UQE_teams_id` ON `teams` (`id`);

CREATE UNIQUE INDEX `UQE_time_entries_id` ON `time_entries` (`id`);

CREATE UNIQUE INDEX `UQE_totp_id` ON `totp` (`id`);

CREATE UNIQUE INDEX `UQE_unsplash_photos_id` ON `unsplash_photos` (`id`);

CREATE UNIQUE INDEX `UQE_user_invite_link_teams_id` ON `user_invite_link_teams` (`id`);

CREATE UNIQUE INDEX `UQE_user_invite_link_teams_invite_team` ON `user_invite_link_teams` (`invite_link_id`,`team_id`);

CREATE UNIQUE INDEX `UQE_user_invite_links_id` ON `user_invite_links` (`id`);

CREATE UNIQUE INDEX `UQE_user_invite_links_token_hash` ON `user_invite_links` (`token_hash`);

CREATE UNIQUE INDEX `UQE_user_tokens_id` ON `user_tokens` (`id`);

CREATE UNIQUE INDEX `UQE_users_id` ON `users` (`id`);

CREATE UNIQUE INDEX `UQE_users_projects_id` ON `users_projects` (`id`);

CREATE UNIQUE INDEX `UQE_users_username` ON `users` (`username`);

CREATE UNIQUE INDEX `UQE_webhooks_id` ON `webhooks` (`id`);
