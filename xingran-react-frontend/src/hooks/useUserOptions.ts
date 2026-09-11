/**
 * useUserOptions hook — fetches enabled user list via React Query with 5min stale time.
 *
 * Multiple components calling `useUserOptions()` share one cache entry (status:0 only).
 * Replaces 5 hand-written useState+useCallback+setState patterns in:
 *   - pages/duty/management
 *   - pages/duty/pools
 *   - pages/duty/schedules/hooks/useScheduleData
 *   - pages/workorder/orders/hooks/useWorkOrderData
 *   - pages/workorder/periodic/templates/hooks/useTemplateData
 */

import { useQuery } from "@tanstack/react-query";
import { getUserList, type SimpleUser } from "@/lib/workorderApi";
import { queryKeys } from "@/lib/queryKeys";

/**
 * Returns the shared enabled-user options list.
 * Data shape matches the existing `SimpleUser[]` consumers expect.
 */
export function useUserOptions() {
  return useQuery<SimpleUser[]>({
    queryKey: queryKeys.user.options,
    queryFn: async () => {
      const result = await getUserList({ status: 0 });
      return result.data?.list ?? [];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}
