<?php
return ['default'=>'stderr','channels'=>['stderr'=>['driver'=>'monolog','level'=>env('LOG_LEVEL','info'),'handler'=>Monolog\Handler\StreamHandler::class,'with'=>['stream'=>'php://stderr']]]];
