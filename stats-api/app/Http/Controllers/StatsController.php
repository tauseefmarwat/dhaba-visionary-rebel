<?php
namespace App\Http\Controllers;
use Illuminate\Support\Facades\Redis;
use Illuminate\Http\JsonResponse;
class StatsController {
    public function index(): JsonResponse {
        $minute = now()->utc()->format('YmdHi');
        $total = (int) Redis::get('dhaba:stats:orders:' . $minute);
        $top = Redis::zrevrange('dhaba:stats:dishes', 0, 4, ['withscores' => true]);
        $dishes = [];
        foreach ($top as $dish => $quantity) { $dishes[] = ['dish'=>(string)$dish, 'quantity'=>(int)$quantity]; }
        return response()->json(['orders_per_minute'=>$total, 'top_dishes'=>$dishes, 'updated_at'=>now()->toISOString()]);
    }
}
