#!/usr/bin/env bash
# ============================================================
# CBT Enterprise — Migration Runner
# Usage:
#   ./migrate.sh up          # Jalankan semua migration baru
#   ./migrate.sh up 003      # Jalankan sampai versi 003
#   ./migrate.sh down 004    # Rollback versi 004
#   ./migrate.sh status      # Lihat status migration
#   ./migrate.sh reset       # Rollback SEMUA (DESTRUCTIVE)
# ============================================================

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
DB_DSN="${DB_DSN:-postgres://cbt_user:cbt_pass@localhost:5432/cbt_db?sslmode=disable}"
MIGRATIONS_DIR="$(dirname "$0")/migrations"

# Parse DSN ke komponen psql
export PGPASSWORD
PGPASSWORD=$(echo "$DB_DSN" | sed -n 's|.*://[^:]*:\([^@]*\)@.*|\1|p')
PGUSER=$(echo    "$DB_DSN" | sed -n 's|.*://\([^:]*\):.*|\1|p')
PGHOST=$(echo    "$DB_DSN" | sed -n 's|.*@\([^:/]*\).*|\1|p')
PGPORT=$(echo    "$DB_DSN" | sed -n 's|.*:\([0-9]*\)/.*|\1|p')
PGDATABASE=$(echo "$DB_DSN" | sed -n 's|.*/\([^?]*\).*|\1|p')

PSQL="psql -h $PGHOST -p ${PGPORT:-5432} -U $PGUSER -d $PGDATABASE"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'

log()  { echo -e "${GREEN}[migrate]${NC} $*"; }
warn() { echo -e "${YELLOW}[migrate]${NC} $*"; }
err()  { echo -e "${RED}[migrate]${NC} $*" >&2; }

# ── Ensure migration table exists ─────────────────────────────────────────────
ensure_runner() {
    $PSQL -q -f "$MIGRATIONS_DIR/000_migration_runner.sql" 2>/dev/null || true
}

# ── Get applied versions ───────────────────────────────────────────────────────
applied_versions() {
    $PSQL -t -c "SELECT version FROM schema_migrations ORDER BY version;" 2>/dev/null \
        | tr -d ' ' | grep -v '^$' || true
}

# ── Status ─────────────────────────────────────────────────────────────────────
cmd_status() {
    ensure_runner
    echo ""
    echo "  Migration Status — CBT Enterprise"
    echo "  ─────────────────────────────────────────"
    printf "  %-6s %-35s %-12s\n" "Ver" "File" "Status"
    echo "  ─────────────────────────────────────────"

    APPLIED=$(applied_versions)

    for f in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
        [ -f "$f" ] || continue
        ver=$(basename "$f" | cut -d_ -f1)
        fname=$(basename "$f")
        if echo "$APPLIED" | grep -q "^${ver}$"; then
            printf "  ${GREEN}%-6s %-35s %-12s${NC}\n" "$ver" "$fname" "✓ applied"
        else
            printf "  ${YELLOW}%-6s %-35s %-12s${NC}\n" "$ver" "$fname" "○ pending"
        fi
    done
    echo ""
}

# ── Up ─────────────────────────────────────────────────────────────────────────
cmd_up() {
    TARGET="${1:-999}"
    ensure_runner
    APPLIED=$(applied_versions)
    COUNT=0

    for f in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
        [ -f "$f" ] || continue
        ver=$(basename "$f" | cut -d_ -f1)

        # Skip if already applied or beyond target
        if echo "$APPLIED" | grep -q "^${ver}$"; then continue; fi
        if [ "$ver" -gt "$TARGET" ] 2>/dev/null; then break; fi

        log "Applying migration $ver: $(basename "$f")..."
        $PSQL -q -f "$f"
        log "  ✓ Done"
        COUNT=$((COUNT + 1))
    done

    if [ "$COUNT" -eq 0 ]; then
        log "No new migrations to apply."
    else
        log "Applied $COUNT migration(s) successfully."
    fi
}

# ── Down ───────────────────────────────────────────────────────────────────────
cmd_down() {
    VERSION="${1:-}"
    if [ -z "$VERSION" ]; then
        err "Usage: $0 down <version>"
        exit 1
    fi

    ROLLBACK_FILE="$MIGRATIONS_DIR/rollback_${VERSION}.sql"
    if [ ! -f "$ROLLBACK_FILE" ]; then
        err "Rollback file not found: $ROLLBACK_FILE"
        exit 1
    fi

    warn "Rolling back migration $VERSION..."
    $PSQL -q -f "$ROLLBACK_FILE"
    log "  ✓ Rollback $VERSION complete"
}

# ── Reset (DESTRUCTIVE) ────────────────────────────────────────────────────────
cmd_reset() {
    warn "⚠️  This will DROP ALL TABLES and data!"
    read -rp "  Type 'yes' to confirm: " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        log "Aborted."
        exit 0
    fi

    log "Running rollback_001.sql (drop all)..."
    $PSQL -q -f "$MIGRATIONS_DIR/rollback_001.sql"
    log "Reset complete. Run './migrate.sh up' to rebuild."
}

# ── Main ───────────────────────────────────────────────────────────────────────
case "${1:-help}" in
    up)     cmd_up     "${2:-}" ;;
    down)   cmd_down   "${2:-}" ;;
    status) cmd_status ;;
    reset)  cmd_reset ;;
    *)
        echo ""
        echo "  Usage: $0 <command> [version]"
        echo ""
        echo "  Commands:"
        echo "    up [version]   Apply all pending migrations (or up to version)"
        echo "    down <version> Roll back a specific migration"
        echo "    status         Show migration status"
        echo "    reset          Drop ALL tables (destructive)"
        echo ""
        echo "  Environment:"
        echo "    DB_DSN=postgres://user:pass@host:port/db"
        echo ""
        ;;
esac
