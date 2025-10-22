-- File com/Access/_errors.lua
--
-- Created at 2025-21-10
-- Description:
--- Centralized error definitions for Access operations
--- to keep API responses consistent and clean.

local errors = {
  -- Common validation
  MISSING_PARAMS = { code = -32602, message = "Missing params" },
  INVALID_FIELD_TYPE = { code = -32602, message = "'fields' must be a non-empty table" },
  INVALID_BY_PARAM = { code = -32602, message = "Invalid 'by' param" },
  NO_VALID_FIELDS = { code = -32604, message = "No valid fields to update" },

  -- Existence / duplication
  UNIT_NOT_FOUND = { code = -32102, message = "Unit is not exists" },
  UNIT_EXISTS = { code = -32101, message = "Unit is already exists" },

  -- Database & constraint
  UNIQUE_CONSTRAINT = { code = -32602, message = "Unique constraint failed" },
  DB_QUERY_FAILED = { code = -32001, message = "Database query failed" },
  DB_EXEC_FAILED = { code = -32002, message = "Database execution failed" },
  DB_INSERT_FAILED = { code = -32003, message = "Failed to create unit" },
  DB_DELETE_FAILED = { code = -32004, message = "Failed to delete unit" },

  -- Generic fallback
  UNKNOWN = { code = -32099, message = "Unexpected internal error" },
}

return errors
