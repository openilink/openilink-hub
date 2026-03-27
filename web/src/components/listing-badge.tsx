import { Badge } from "@/components/ui/badge";

export function ListingBadge({ listing }: { listing?: string }) {
  if (listing === "listed") return <Badge variant="default">已上架</Badge>;
  if (listing === "pending")
    return (
      <Badge variant="outline" className="text-orange-500 border-orange-500">
        审核中
      </Badge>
    );
  if (listing === "rejected") return <Badge variant="destructive">已拒绝</Badge>;
  return <Badge variant="secondary">未上架</Badge>;
}
