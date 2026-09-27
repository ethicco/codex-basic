import { NextRequest } from "next/server";
import { handleRefresh } from "@/lib/auth-proxy";

export async function POST(request: NextRequest) {
  return handleRefresh(request);
}
