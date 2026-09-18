CREATE SCHEMA basestack_rbac;
CREATE TABLE basestack_rbac.roles (name text PRIMARY KEY);
CREATE TABLE basestack_rbac.permissions (
 role text NOT NULL REFERENCES basestack_rbac.roles(name) ON DELETE CASCADE,
 permission text NOT NULL,
 PRIMARY KEY (role,permission)
);
CREATE TABLE basestack_rbac.user_roles (
 user_id uuid NOT NULL REFERENCES basestack_auth.users(id) ON DELETE CASCADE,
 role text NOT NULL REFERENCES basestack_rbac.roles(name) ON DELETE CASCADE,
 PRIMARY KEY (user_id,role)
);
-- Names are conveniences, never implicit superuser bypasses.
INSERT INTO basestack_rbac.roles(name) VALUES ('admin'),('member');
