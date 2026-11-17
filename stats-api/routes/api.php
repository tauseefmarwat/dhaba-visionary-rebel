<?php
use Illuminate\Support\Facades\Route;
use App\Http\Controllers\StatsController;
Route::get('/stats', [StatsController::class, 'index']);
Route::get('/health', fn () => response()->json(['status'=>'ok','service'=>'stats-api']));
