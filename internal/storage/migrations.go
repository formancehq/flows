package storage

import (
	"context"

	"github.com/formancehq/go-libs/v2/migrations"
	"github.com/uptrace/bun"
)

var _migrations = []migrations.Migration{
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					create table "workflows" (
						config jsonb,
						id varchar not null,
						created_at timestamp default now(),
						updated_at timestamp default now(),
						primary key (id)
					);
					create table "workflow_instances" (
						workflow_id varchar references workflows (id),
						id varchar,
						created_at timestamp default now(),
						updated_at timestamp default now(),
						primary key (id)
					);
					create table "workflow_instance_stage_statuses" (
						instance_id varchar references workflow_instances (id),
						stage int,
						started_at timestamp default now(),
						terminated_at timestamp default null,
						error varchar,
						primary key (instance_id, stage)
					);
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					alter table "workflow_instances" add column terminated bool;
					alter table "workflow_instances" add column terminated_at timestamp default null;
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					alter table "workflow_instances" add column error varchar;
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					alter table "workflows" add column if not exists deleted_at timestamp default null;
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					create table triggers (
					    id varchar primary key,
					    workflow_id varchar references workflows(id),
					    filter varchar null,
					    event varchar not null,
					    vars jsonb,
					    created_at timestamp not null default now(),
					    deleted_at timestamp default null
					);
					create table triggers_occurrences (
					    workflow_instance_id varchar references workflow_instances(id),
					    trigger_id varchar references triggers(id),
					    event_id varchar not null,
					    date timestamp not null default now(),
					    event jsonb not null,
					    primary key (trigger_id, event_id)
					);
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
					alter table "workflow_instance_stage_statuses" 
					drop constraint workflow_instance_stage_statuses_pkey;
					
					alter table "workflow_instance_stage_statuses"
					add column temporal_run_id varchar;
					
					update "workflow_instance_stage_statuses"
					set temporal_run_id = '';
					
					alter table "workflow_instance_stage_statuses"
					add primary key (instance_id, stage, temporal_run_id);
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
				alter table "triggers_occurrences"
				add column error varchar;
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
				alter table "triggers_occurrences" 
				drop constraint triggers_occurrences_pkey;

				alter table "triggers_occurrences"
				drop column event_id;

				alter table "triggers_occurrences"
				add column id varchar;

				update "triggers_occurrences"
				set id = gen_random_uuid();

				alter table "triggers_occurrences"
				add primary key (id);
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
				alter table "triggers"
				add column name varchar;
				`); err != nil {
				return err
			}
			return nil
		},
	},
	{
		Up: func(ctx context.Context, tx bun.IDB) error {
			if _, err := tx.ExecContext(ctx, `
				alter table "triggers"
				add column version varchar default NULL;
				`); err != nil {
				return err
			}
			return nil
		},
	},
}

func Migrate(ctx context.Context, db *bun.DB) error {
	return MigrateUntil(ctx, db, len(_migrations))
}

func MigrateUntil(ctx context.Context, db *bun.DB, until int) error {
	migrator := migrations.NewMigrator(db)
	migrator.RegisterMigrations(_migrations[:until]...)

	return migrator.Up(ctx)
}
