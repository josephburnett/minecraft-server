/**
 * Permission checking helpers for player abilities
 */

/**
 * Check if a player has operator privileges
 * @param {Player} player
 * @returns {boolean}
 */
export function isOperator(player) {
    try {
        player.runCommand("testfor @s[tag=__op_check__]");
        return true;
    } catch (e) {
        return !e.message?.includes("permission");
    }
}

/**
 * Check if a player has permission to use an ability
 * @param {Player} player
 * @param {object} ability - Ability config with permission field
 * @returns {boolean}
 */
export function hasPermission(player, ability) {
    if (ability.permission === "disabled") return false;
    if (ability.permission === "everyone") return true;
    if (ability.permission === "operator") return isOperator(player);
    return false;
}
