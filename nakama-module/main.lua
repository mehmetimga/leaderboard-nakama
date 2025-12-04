--[[
  Nakama runtime module for leaderboard initialization
]]--

local nk = require("nakama")

-- Initialize leaderboards on server startup
local function init_leaderboards()
    nk.logger_info("Creating leaderboards...")
    
    -- Create global_scores leaderboard
    local id = "global_scores"
    local authoritative = false  -- Allow client writes
    local sort = "desc"          -- Higher score is better
    local operator = "best"      -- Keep best score
    local reset = ""             -- No reset schedule
    local metadata = {}
    
    local success, err = pcall(function()
        nk.leaderboard_create(id, authoritative, sort, operator, reset, metadata)
    end)
    
    if success then
        nk.logger_info("Created leaderboard: " .. id)
    else
        nk.logger_warn("Leaderboard may already exist: " .. id)
    end
    
    -- Create weekly_scores leaderboard
    id = "weekly_scores"
    reset = "0 0 * * 0"  -- Reset every Sunday at midnight
    
    success, err = pcall(function()
        nk.leaderboard_create(id, authoritative, sort, operator, reset, metadata)
    end)
    
    if success then
        nk.logger_info("Created leaderboard: " .. id)
    else
        nk.logger_warn("Leaderboard may already exist: " .. id)
    end
    
    nk.logger_info("Leaderboard initialization complete")
end

-- Run initialization
init_leaderboards()

nk.logger_info("Nakama module loaded successfully")

