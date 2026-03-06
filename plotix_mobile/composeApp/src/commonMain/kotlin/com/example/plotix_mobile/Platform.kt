package com.example.plotix_mobile

interface Platform {
    val name: String
}

expect fun getPlatform(): Platform