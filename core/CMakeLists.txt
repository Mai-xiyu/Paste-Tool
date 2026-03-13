add_library(paste_core STATIC app_core.c)
target_include_directories(paste_core PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
