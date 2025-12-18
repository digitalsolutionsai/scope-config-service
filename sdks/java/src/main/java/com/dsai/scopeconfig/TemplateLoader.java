package com.dsai.scopeconfig;

import vn.dsai.config.v1.*;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.util.*;
import java.util.logging.Logger;
import java.util.stream.Collectors;

import org.yaml.snakeyaml.Yaml;

/**
 * Utility class for loading configuration templates from YAML files.
 * 
 * Simply place your template files in a directory and use this class to
 * automatically load and apply them to the config service.
 * 
 * <pre>{@code
 * try (ConfigClient client = ConfigClient.fromEnvironment().build()) {
 *     TemplateLoader.loadFromDir(client, "./templates", "system");
 * }
 * }</pre>
 */
public class TemplateLoader {
    
    private static final Logger logger = Logger.getLogger(TemplateLoader.class.getName());
    
    private TemplateLoader() {
        // Utility class
    }
    
    /**
     * Load and apply all YAML templates from a directory.
     * 
     * @param client The connected ConfigClient instance
     * @param dirPath Path to the templates directory
     * @param user The user performing the action
     * @throws IOException If there's an error reading template files
     * @throws ConfigServiceException If there's an error applying templates
     */
    public static void loadFromDir(ConfigClient client, String dirPath, String user) throws IOException {
        File dir = new File(dirPath);
        
        if (!dir.exists() || !dir.isDirectory()) {
            logger.info("Templates directory " + dirPath + " does not exist, skipping template import");
            return;
        }
        
        File[] yamlFiles = dir.listFiles((d, name) -> 
            name.endsWith(".yaml") || name.endsWith(".yml"));
        
        if (yamlFiles == null || yamlFiles.length == 0) {
            logger.info("No template files found in " + dirPath);
            return;
        }
        
        logger.info("Found " + yamlFiles.length + " template file(s) to import");
        
        for (File file : yamlFiles) {
            loadAndApplyTemplateFile(client, file, user);
        }
    }
    
    /**
     * Load and apply a single YAML template file.
     * 
     * @param client The connected ConfigClient instance
     * @param filePath Path to the template file
     * @param user The user performing the action
     * @throws IOException If there's an error reading the file
     * @throws ConfigServiceException If there's an error applying the template
     */
    public static void loadFromFile(ConfigClient client, String filePath, String user) throws IOException {
        loadAndApplyTemplateFile(client, new File(filePath), user);
    }
    
    private static void loadAndApplyTemplateFile(ConfigClient client, File file, String user) throws IOException {
        Yaml yaml = new Yaml();
        Map<String, Object> data;
        
        try (FileInputStream fis = new FileInputStream(file)) {
            data = yaml.load(fis);
        }
        
        if (data == null) {
            logger.warning("Empty template file: " + file.getName());
            return;
        }
        
        // Validate required fields
        @SuppressWarnings("unchecked")
        Map<String, Object> service = (Map<String, Object>) data.get("service");
        if (service == null || !service.containsKey("id")) {
            throw new ConfigServiceException("Template file " + file.getName() + " missing 'service.id'", 
                io.grpc.Status.INVALID_ARGUMENT);
        }
        
        String serviceName = (String) service.get("id");
        String serviceLabel = (String) service.getOrDefault("label", serviceName);
        
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> groups = (List<Map<String, Object>>) data.get("groups");
        if (groups == null || groups.isEmpty()) {
            logger.warning("No groups defined in template: " + file.getName());
            return;
        }
        
        for (Map<String, Object> group : groups) {
            applyGroupTemplate(client, serviceName, serviceLabel, group, user);
            logger.info("Successfully imported template: service=" + serviceName + 
                       ", group=" + group.get("id") + " from " + file.getName());
        }
    }
    
    private static void applyGroupTemplate(
            ConfigClient client, 
            String serviceName, 
            String serviceLabel, 
            Map<String, Object> group, 
            String user) {
        
        String groupId = (String) group.getOrDefault("id", "");
        String groupLabel = (String) group.getOrDefault("label", groupId);
        String groupDescription = (String) group.getOrDefault("description", "");
        int sortOrder = ((Number) group.getOrDefault("sortOrder", 0)).intValue();
        
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> fieldMaps = (List<Map<String, Object>>) group.getOrDefault("fields", Collections.emptyList());
        
        List<ConfigFieldTemplate> fields = fieldMaps.stream()
            .map(TemplateLoader::buildFieldTemplate)
            .collect(Collectors.toList());
        
        ConfigIdentifier identifier = ConfigIdentifier.newBuilder()
            .setServiceName(serviceName)
            .setGroupId(groupId)
            .build();
        
        ConfigTemplate template = ConfigTemplate.newBuilder()
            .setIdentifier(identifier)
            .setServiceLabel(serviceLabel)
            .setGroupLabel(groupLabel)
            .setGroupDescription(groupDescription)
            .addAllFields(fields)
            .setSortOrder(sortOrder)
            .build();
        
        client.applyConfigTemplate(template, user);
    }
    
    private static ConfigFieldTemplate buildFieldTemplate(Map<String, Object> f) {
        ConfigFieldTemplate.Builder builder = ConfigFieldTemplate.newBuilder()
            .setPath((String) f.getOrDefault("path", ""))
            .setLabel((String) f.getOrDefault("label", ""))
            .setDescription((String) f.getOrDefault("description", ""))
            .setType(toFieldType((String) f.getOrDefault("type", "STRING")))
            .setDefaultValue((String) f.getOrDefault("defaultValue", ""))
            .setSortOrder(((Number) f.getOrDefault("sortOrder", 0)).intValue());
        
        @SuppressWarnings("unchecked")
        List<String> displayOn = (List<String>) f.getOrDefault("displayOn", Collections.emptyList());
        builder.addAllDisplayOn(displayOn.stream()
            .map(TemplateLoader::toScope)
            .collect(Collectors.toList()));
        
        @SuppressWarnings("unchecked")
        List<Map<String, String>> options = (List<Map<String, String>>) f.getOrDefault("options", Collections.emptyList());
        builder.addAllOptions(options.stream()
            .map(o -> ValueOption.newBuilder()
                .setValue(o.getOrDefault("value", ""))
                .setLabel(o.getOrDefault("label", o.getOrDefault("value", "")))
                .build())
            .collect(Collectors.toList()));
        
        return builder.build();
    }
    
    private static Scope toScope(String s) {
        switch (s.toUpperCase()) {
            case "SYSTEM": return Scope.SYSTEM;
            case "PROJECT": return Scope.PROJECT;
            case "STORE": return Scope.STORE;
            case "USER": return Scope.USER;
            default: return Scope.SCOPE_UNSPECIFIED;
        }
    }
    
    private static FieldType toFieldType(String t) {
        switch (t.toUpperCase()) {
            case "STRING": return FieldType.STRING;
            case "INT": return FieldType.INT;
            case "FLOAT": return FieldType.FLOAT;
            case "BOOLEAN": return FieldType.BOOLEAN;
            case "JSON": return FieldType.JSON;
            case "ARRAY_STRING": return FieldType.ARRAY_STRING;
            case "SECRET": return FieldType.SECRET;
            default: return FieldType.STRING;
        }
    }
}
